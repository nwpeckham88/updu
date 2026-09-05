package p2p

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/realtime"
	"github.com/updu/updu/internal/storage"
)

// SyncResponse is the payload returned by /api/v1/p2p/feed or /api/v1/p2p/sync.
type SyncResponse struct {
	NodeID    string                `json:"node_id"`
	Name      string                `json:"name"`
	Telemetry *models.PeerTelemetry `json:"telemetry"`
	Monitors  []*models.Monitor     `json:"monitors"`
	Timestamp int64                 `json:"timestamp"`
}

// Manager coordinates P2P discovery, pairing, telemetry, federated monitoring, and survivor triage.
type Manager struct {
	identity  *NodeIdentity
	discovery *Discovery
	db        *storage.DB
	sse       *realtime.Hub
	client    *http.Client
	localAddr string // host:port this node is listening on, advertised in pair-requests

	stop chan struct{}
	wg   sync.WaitGroup

	mu             sync.RWMutex
	peerMonitors   map[string][]*models.Monitor
	peerTelemetry  map[string]*models.PeerTelemetry
	peerTriage     map[string]*models.SurvivorTriage
	peerLastOnline map[string]time.Time
}

// isInternalAddr checks if a host/address belongs to Tailscale, RFC1918 private subnets, loopback, or tailnet domain.
func isInternalAddr(address string) bool {
	host := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(address), "http://"), "https://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.ToLower(strings.TrimSpace(host))

	if host == "localhost" || strings.HasSuffix(host, ".local") || strings.Contains(host, "tailnet") {
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() {
			return true
		}
		tailscaleSubnet := &net.IPNet{
			IP:   net.ParseIP("100.64.0.0"),
			Mask: net.CIDRMask(10, 32),
		}
		if tailscaleSubnet.Contains(ip) {
			return true
		}
	}

	return false
}

// ResolveAdvertisedAddress returns the best reachable host:port address for this node.
// Priority:
// 1. UPDU_ADVERTISED_ADDRESS env var (e.g. "100.64.0.1:3000")
// 2. If host is already a specific routable IP (not 0.0.0.0, 127.0.0.1, empty, localhost), use host:port
// 3. First IP found in 100.64.0.0/10 (Tailscale CGNAT)
// 4. First private IPv4 (RFC 1918: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
// 5. Fallback: 127.0.0.1:port
func ResolveAdvertisedAddress(host string, port int) string {
	if envAddr := strings.TrimSpace(os.Getenv("UPDU_ADVERTISED_ADDRESS")); envAddr != "" {
		clean := strings.TrimPrefix(strings.TrimPrefix(envAddr, "http://"), "https://")
		return clean
	}

	if host != "" && host != "0.0.0.0" && host != "127.0.0.1" && host != "::" && host != "localhost" {
		return fmt.Sprintf("%s:%d", host, port)
	}

	tailscaleSubnet := &net.IPNet{
		IP:   net.ParseIP("100.64.0.0"),
		Mask: net.CIDRMask(10, 32),
	}

	var fallbackPrivateIP string

	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip == nil || ip.To4() == nil || ip.IsLoopback() {
					continue
				}
				ipv4 := ip.To4()
				if tailscaleSubnet.Contains(ipv4) {
					return fmt.Sprintf("%s:%d", ipv4.String(), port)
				}
				if fallbackPrivateIP == "" && (ipv4.IsPrivate()) {
					fallbackPrivateIP = ipv4.String()
				}
			}
		}
	}

	if fallbackPrivateIP != "" {
		return fmt.Sprintf("%s:%d", fallbackPrivateIP, port)
	}

	return fmt.Sprintf("127.0.0.1:%d", port)
}

// NewManager creates a new P2P Manager.
// localAddr is the "host:port" this node listens on and will advertise in pair-requests.
func NewManager(id *NodeIdentity, disc *Discovery, db *storage.DB, sse *realtime.Hub, localAddr string) *Manager {
	return &Manager{
		identity:       id,
		discovery:      disc,
		db:             db,
		sse:            sse,
		client:         &http.Client{Timeout: 5 * time.Second},
		localAddr:      localAddr,
		stop:           make(chan struct{}),
		peerMonitors:   make(map[string][]*models.Monitor),
		peerTelemetry:  make(map[string]*models.PeerTelemetry),
		peerTriage:     make(map[string]*models.SurvivorTriage),
		peerLastOnline: make(map[string]time.Time),
	}
}


// Start begins background discovery and peer federation polling.
func (m *Manager) Start(ctx context.Context) error {
	if m.discovery != nil {
		if err := m.discovery.Start(); err != nil {
			slog.Warn("discovery failed to start", "error", err)
		}
	}

	m.wg.Add(1)
	go m.pollLoop()

	slog.Info("P2P federation manager started", "node_id", m.identity.NodeID)
	return nil
}

// Stop terminates discovery and peer federation polling.
func (m *Manager) Stop() {
	close(m.stop)
	if m.discovery != nil {
		m.discovery.Stop()
	}
	m.wg.Wait()
}

// Identity returns the local node identity.
func (m *Manager) Identity() *NodeIdentity {
	return m.identity
}

// Discovery returns the discovery service.
func (m *Manager) Discovery() *Discovery {
	return m.discovery
}

// pollLoop periodically polls approved peers for monitor updates and telemetry.
func (m *Manager) pollLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(6 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			m.pollApprovedPeers()
		}
	}
}

func (m *Manager) pollApprovedPeers() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	peers, err := m.db.ListApprovedPeers(ctx)
	if err != nil {
		return
	}

	for _, peer := range peers {
		m.pollPeer(ctx, peer)
	}
}

func (m *Manager) pollPeer(ctx context.Context, peer *models.Peer) {
	cleanAddr := strings.TrimPrefix(strings.TrimPrefix(peer.Address, "http://"), "https://")
	scheme := "http://"
	if strings.HasPrefix(peer.Address, "https://") && !isInternalAddr(cleanAddr) {
		scheme = "https://"
	}
	url := fmt.Sprintf("%s%s/api/v1/p2p/feed", scheme, cleanAddr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		m.handlePeerFailure(peer)
		return
	}

	// Cryptographic auth headers
	ts := time.Now().Unix()
	msg := fmt.Sprintf("%s:%d", m.identity.NodeID, ts)
	sig := m.identity.Sign([]byte(msg))

	req.Header.Set("X-Peer-ID", m.identity.NodeID)
	req.Header.Set("X-Peer-Timestamp", strconv.FormatInt(ts, 10))
	req.Header.Set("X-Peer-Signature", sig)

	resp, err := m.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			_ = resp.Body.Close()
		}
		m.handlePeerFailure(peer)
		return
	}
	defer resp.Body.Close()

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		m.handlePeerFailure(peer)
		return
	}

	// Success: peer is online
	now := time.Now()
	m.mu.Lock()
	m.peerLastOnline[peer.ID] = now
	if syncResp.Telemetry != nil {
		m.peerTelemetry[peer.ID] = syncResp.Telemetry
	}

	// Tag remote monitors with peer group and metadata
	groupName := fmt.Sprintf("☁️ %s", peer.Name)
	for i := range syncResp.Monitors {
		syncResp.Monitors[i].Groups = []string{groupName}
		syncResp.Monitors[i].IsFederated = true
		syncResp.Monitors[i].PeerID = peer.ID
		syncResp.Monitors[i].PeerName = peer.Name
		syncResp.Monitors[i].PeerAddress = peer.Address
	}
	m.peerMonitors[peer.ID] = syncResp.Monitors

	// Clear any active triage for this peer
	hadTriage := m.peerTriage[peer.ID] != nil
	delete(m.peerTriage, peer.ID)
	m.mu.Unlock()

	// Update DB last_seen and telemetry metadata
	var metaJSON []byte
	if syncResp.Telemetry != nil {
		metaJSON, _ = json.Marshal(syncResp.Telemetry)
	}
	_ = m.db.UpdatePeerLastSeen(ctx, peer.ID, now, metaJSON)

	// Broadcast peer update if triage was cleared or monitors updated
	if m.sse != nil {
		if hadTriage {
			m.sse.Broadcast(realtime.Event{
				Type: "peer_recovered",
				Data: map[string]string{
					"peer_id": peer.ID,
					"name":    peer.Name,
				},
			})
		}
		for _, mon := range syncResp.Monitors {
			m.sse.Broadcast(realtime.Event{
				Type: "monitor:status",
				Data: map[string]any{
					"id":              "fed_" + mon.ID,
					"status":          mon.Status,
					"last_latency_ms": mon.LastLatency,
					"last_check":      mon.LastCheck,
					"recent_checks":   mon.RecentChecks,
					"uptime_24h":      mon.Uptime24h,
				},
			})
		}
	}
}

func (m *Manager) handlePeerFailure(peer *models.Peer) {
	m.mu.Lock()
	lastOnline, existed := m.peerLastOnline[peer.ID]
	lastTelem := m.peerTelemetry[peer.ID]
	alreadyTriaged := m.peerTriage[peer.ID] != nil
	m.mu.Unlock()

	// If never seen or triage already emitted, don't run diagnostic repeatedly
	if alreadyTriaged {
		return
	}

	// Run survivor triage autopsy
	triage := RunTriage(context.Background(), peer, lastTelem)

	m.mu.Lock()
	m.peerTriage[peer.ID] = triage
	m.mu.Unlock()

	slog.Warn("peer connection lost - triage generated",
		"peer_id", peer.ID,
		"peer_name", peer.Name,
		"cause", triage.ProbableCause,
		"last_online", lastOnline,
		"existed", existed,
	)

	// Broadcast triage report to UI via SSE
	if m.sse != nil {
		m.sse.Broadcast(realtime.Event{
			Type: "peer_triage",
			Data: triage,
		})
	}
}

// HandlePairProposal validates a cryptographic pairing request from another node.
func (m *Manager) HandlePairProposal(ctx context.Context, prop *models.PairProposal) error {
	if prop.NodeID == "" || prop.PublicKey == "" || prop.Signature == "" {
		return errors.New("invalid pair proposal: missing required fields")
	}

	// Prevent self-pairing
	if prop.NodeID == m.identity.NodeID {
		return errors.New("cannot pair with self")
	}

	// Verify timestamp is within 10 minutes to prevent replays
	now := time.Now().Unix()
	if prop.Timestamp < now-600 || prop.Timestamp > now+600 {
		return errors.New("pair proposal expired or timestamp skewed")
	}

	// Verify signature
	canonical := fmt.Sprintf("%s:%s:%s:%d", prop.NodeID, prop.Name, prop.Address, prop.Timestamp)
	if !VerifySignature(prop.PublicKey, []byte(canonical), prop.Signature) {
		return errors.New("cryptographic signature verification failed")
	}

	// Check if already approved
	existing, err := m.db.GetPeer(ctx, prop.NodeID)
	status := models.PeerStatusPending
	if err == nil && existing != nil && existing.Status == models.PeerStatusApproved {
		status = models.PeerStatusApproved
	}

	cleanAddr := strings.TrimPrefix(strings.TrimPrefix(prop.Address, "http://"), "https://")
	peer := &models.Peer{
		ID:        prop.NodeID,
		Name:      prop.Name,
		Address:   cleanAddr,
		PublicKey: prop.PublicKey,
		Role:      models.PeerRolePeer,
		Status:    status,
	}

	if err := m.db.CreatePeer(ctx, peer); err != nil {
		return fmt.Errorf("storing peer: %w", err)
	}

	slog.Info("received valid pairing proposal", "peer_id", prop.NodeID, "name", prop.Name, "status", status)

	if m.sse != nil {
		m.sse.Broadcast(realtime.Event{
			Type: "peer_proposal",
			Data: peer,
		})
	}

	return nil
}

// ConnectPeer initiates pairing with a remote node by address (e.g. Tailnet IP or hostname).
func (m *Manager) ConnectPeer(ctx context.Context, address string, name string) (*models.Peer, error) {
	cleanAddr := strings.TrimPrefix(strings.TrimPrefix(address, "http://"), "https://")
	scheme := "http://"
	if strings.HasPrefix(address, "https://") && !isInternalAddr(cleanAddr) {
		scheme = "https://"
	}
	idURL := fmt.Sprintf("%s%s/api/v1/p2p/identity", scheme, cleanAddr)

	req, err := http.NewRequestWithContext(ctx, "GET", idURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to remote peer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote peer returned status %d", resp.StatusCode)
	}

	var remoteID struct {
		NodeID    string `json:"node_id"`
		Name      string `json:"name"`
		PublicKey string `json:"public_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&remoteID); err != nil {
		return nil, fmt.Errorf("decoding remote identity: %w", err)
	}

	if remoteID.NodeID == m.identity.NodeID {
		return nil, errors.New("cannot connect to self")
	}

	peerName := remoteID.Name
	if name != "" {
		peerName = name
	}

	// Prepare signed pair request — advertise our real listen address
	ts := time.Now().Unix()
	canonical := fmt.Sprintf("%s:%s:%s:%d", m.identity.NodeID, m.identity.Name, m.localAddr, ts)
	sig := m.identity.Sign([]byte(canonical))

	proposal := models.PairProposal{
		NodeID:    m.identity.NodeID,
		Name:      m.identity.Name,
		Address:   m.localAddr,
		PublicKey: m.identity.PubKeyHex,
		Signature: sig,
		Timestamp: ts,
	}

	propData, _ := json.Marshal(proposal)
	pairURL := fmt.Sprintf("%s%s/api/v1/p2p/pair-request", scheme, cleanAddr)
	pairReq, _ := http.NewRequestWithContext(ctx, "POST", pairURL, bytes.NewReader(propData))
	pairReq.Header.Set("Content-Type", "application/json")

	pairResp, err := m.client.Do(pairReq)
	if err != nil {
		slog.Warn("reciprocal pair request failed", "error", err)
	} else {
		_ = pairResp.Body.Close()
	}

	// Store locally as PENDING — the local admin must explicitly approve before
	// data flows in either direction. Connecting is an invitation, not a grant.
	peer := &models.Peer{
		ID:        remoteID.NodeID,
		Name:      peerName,
		Address:   cleanAddr,
		PublicKey: remoteID.PublicKey,
		Role:      models.PeerRolePeer,
		Status:    models.PeerStatusPending,
	}

	if err := m.db.CreatePeer(ctx, peer); err != nil {
		return nil, fmt.Errorf("saving peer: %w", err)
	}

	slog.Info("peer connection initiated — awaiting local approval", "peer_id", peer.ID, "name", peer.Name, "address", peer.Address)
	return peer, nil
}

// ApprovePeer approves a pending pairing proposal and marks it approved.
func (m *Manager) ApprovePeer(ctx context.Context, id string) error {
	peer, err := m.db.GetPeer(ctx, id)
	if err != nil || peer == nil {
		return errors.New("peer not found")
	}

	if err := m.db.UpdatePeerStatus(ctx, id, models.PeerStatusApproved); err != nil {
		return err
	}

	// Asynchronously notify peer of reciprocal approval — advertise our real listen address
	go func() {
		cleanAddr := strings.TrimPrefix(strings.TrimPrefix(peer.Address, "http://"), "https://")
		scheme := "http://"
		if strings.HasPrefix(peer.Address, "https://") && !isInternalAddr(cleanAddr) {
			scheme = "https://"
		}
		ts := time.Now().Unix()
		canonical := fmt.Sprintf("%s:%s:%s:%d", m.identity.NodeID, m.identity.Name, m.localAddr, ts)
		sig := m.identity.Sign([]byte(canonical))

		proposal := models.PairProposal{
			NodeID:    m.identity.NodeID,
			Name:      m.identity.Name,
			Address:   m.localAddr,
			PublicKey: m.identity.PubKeyHex,
			Signature: sig,
			Timestamp: ts,
		}
		data, _ := json.Marshal(proposal)
		url := fmt.Sprintf("%s%s/api/v1/p2p/pair-request", scheme, cleanAddr)
		req, _ := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	return nil
}

// RejectPeer rejects or removes a peer.
func (m *Manager) RejectPeer(ctx context.Context, id string) error {
	return m.db.DeletePeer(ctx, id)
}

// ListPeers returns all configured peers populated with active runtime status and telemetry.
func (m *Manager) ListPeers(ctx context.Context) ([]*models.Peer, error) {
	peers, err := m.db.ListPeers(ctx)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range peers {
		if triage, ok := m.peerTriage[p.ID]; ok {
			p.Status = models.PeerStatusDisconnected
			p.Triage = triage
		} else if _, online := m.peerLastOnline[p.ID]; online && p.Status == models.PeerStatusApproved {
			// Online
		}
		if telem, ok := m.peerTelemetry[p.ID]; ok {
			p.Metadata, _ = json.Marshal(telem)
		}
		if mons, ok := m.peerMonitors[p.ID]; ok {
			p.Monitors = mons
		}
	}

	return peers, nil
}

// GetFederatedMonitors returns all monitors collected from approved online peers.
func (m *Manager) GetFederatedMonitors() []*models.Monitor {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []*models.Monitor
	for _, mons := range m.peerMonitors {
		all = append(all, mons...)
	}
	return all
}

// GetFederatedMonitor finds a specific federated monitor by ID or fed_ prefixed ID.
func (m *Manager) GetFederatedMonitor(id string) *models.Monitor {
	cleanID := strings.TrimPrefix(id, "fed_")
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, mons := range m.peerMonitors {
		for _, mon := range mons {
			if mon.ID == cleanID || mon.ID == id {
				return mon
			}
		}
	}
	return nil
}

// ListActiveTriage returns all currently active survivor triage reports.
func (m *Manager) ListActiveTriage() []*models.SurvivorTriage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*models.SurvivorTriage
	for _, tr := range m.peerTriage {
		list = append(list, tr)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].DiscoveredAt.After(list[j].DiscoveredAt)
	})
	return list
}

// AuthenticatePeer verifies request headers against an approved peer's public key.
func (m *Manager) AuthenticatePeer(ctx context.Context, peerID, timestampStr, sigHex string) bool {
	peer, err := m.db.GetPeer(ctx, peerID)
	if err != nil || peer == nil || peer.Status != models.PeerStatusApproved {
		return false
	}

	ts, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false
	}

	// Verify within 10 minute skew window
	now := time.Now().Unix()
	if ts < now-600 || ts > now+600 {
		return false
	}

	msg := fmt.Sprintf("%s:%d", peerID, ts)
	return VerifySignature(peer.PublicKey, []byte(msg), sigHex)
}
