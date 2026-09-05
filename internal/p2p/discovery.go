package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/version"
)

const (
	DefaultBeaconPort = 3001
	BeaconInterval    = 5 * time.Second
	PeerExpiry        = 60 * time.Second
)

// BeaconPacket represents the broadcast message on LAN.
type BeaconPacket struct {
	NodeID  string `json:"node_id"`
	Name    string `json:"name"`
	Port    int    `json:"port"`
	Version string `json:"version"`
}

// Discovery manages UDP beacon broadcasting and listening for LAN and Tailnet discovery.
type Discovery struct {
	nodeID   string
	name     string
	httpPort int
	udpPort  int

	conn *net.UDPConn
	stop chan struct{}
	wg   sync.WaitGroup

	mu              sync.RWMutex
	discovered      map[string]*models.DiscoveredPeer
	lastPeerRefresh time.Time
	cachedPeerIPs   []string
}

// NewDiscovery creates a new P2P discovery service.
func NewDiscovery(nodeID, name string, httpPort, udpPort int) *Discovery {
	if udpPort <= 0 {
		udpPort = DefaultBeaconPort
	}
	return &Discovery{
		nodeID:     nodeID,
		name:       name,
		httpPort:   httpPort,
		udpPort:    udpPort,
		stop:       make(chan struct{}),
		discovered: make(map[string]*models.DiscoveredPeer),
	}
}

// Start begins the UDP beacon listener and broadcaster.
func (d *Discovery) Start() error {
	addr := &net.UDPAddr{Port: d.udpPort}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		slog.Warn("failed to bind UDP beacon listener (P2P discovery listener disabled)", "port", d.udpPort, "error", err)
		return nil
	}
	d.conn = conn

	d.wg.Add(2)
	go d.listenLoop()
	go d.broadcastLoop()

	slog.Info("P2P discovery beacon started", "port", d.udpPort)
	return nil
}

// Stop terminates the beacon loops and closes the UDP socket.
func (d *Discovery) Stop() {
	close(d.stop)
	if d.conn != nil {
		d.conn.Close()
	}
	d.wg.Wait()
}

func (d *Discovery) broadcastLoop() {
	defer d.wg.Done()
	ticker := time.NewTicker(BeaconInterval)
	defer ticker.Stop()

	// Initial broadcast
	d.sendBroadcast()

	for {
		select {
		case <-d.stop:
			return
		case <-ticker.C:
			d.sendBroadcast()
		}
	}
}

func (d *Discovery) sendBroadcast() {
	packet := BeaconPacket{
		NodeID:  d.nodeID,
		Name:    d.name,
		Port:    d.httpPort,
		Version: version.Version,
	}

	data, err := json.Marshal(packet)
	if err != nil {
		return
	}

	// 1. Send broadcast to LAN
	sendBeaconTo(data, &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: d.udpPort,
	})

	// 2. Send unicast beacon to discovered Tailnet peers & configured targets
	for _, target := range d.getPeerTargets() {
		sendBeaconTo(data, target)
	}
}

func sendBeaconTo(data []byte, addr *net.UDPAddr) {
	if addr == nil || addr.IP == nil {
		return
	}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()
	_, _ = conn.Write(data)
}

func (d *Discovery) getPeerTargets() []*net.UDPAddr {
	var addrs []*net.UDPAddr
	seen := make(map[string]bool)

	// Refresh Tailscale peers every 30 seconds
	d.mu.Lock()
	if time.Since(d.lastPeerRefresh) > 30*time.Second {
		d.cachedPeerIPs = queryTailscalePeers()
		d.lastPeerRefresh = time.Now()
	}
	tailscaleIPs := append([]string(nil), d.cachedPeerIPs...)
	d.mu.Unlock()

	for _, ipStr := range tailscaleIPs {
		if !seen[ipStr] {
			seen[ipStr] = true
			if ip := net.ParseIP(ipStr); ip != nil {
				addrs = append(addrs, &net.UDPAddr{IP: ip, Port: d.udpPort})
			}
		}
	}

	// Configured discovery targets from environment (e.g. "100.64.0.1:3001,100.64.0.4")
	targetsEnv := os.Getenv("UPDU_DISCOVERY_TARGETS")
	if targetsEnv == "" {
		targetsEnv = os.Getenv("UPDU_PEER_TARGETS")
	}
	if targetsEnv != "" {
		for _, raw := range strings.Split(targetsEnv, ",") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			host := raw
			port := d.udpPort
			if h, p, err := net.SplitHostPort(raw); err == nil {
				host = h
				if parsedPort, err := strconv.Atoi(p); err == nil && parsedPort > 0 {
					port = parsedPort
				}
			}
			if ip := net.ParseIP(host); ip != nil {
				key := fmt.Sprintf("%s:%d", ip.String(), port)
				if !seen[key] {
					seen[key] = true
					addrs = append(addrs, &net.UDPAddr{IP: ip, Port: port})
				}
			} else {
				// Resolve hostname if needed
				if resolvedIPs, err := net.LookupIP(host); err == nil {
					for _, rip := range resolvedIPs {
						if ipv4 := rip.To4(); ipv4 != nil {
							key := fmt.Sprintf("%s:%d", ipv4.String(), port)
							if !seen[key] {
								seen[key] = true
								addrs = append(addrs, &net.UDPAddr{IP: ipv4, Port: port})
							}
							break
						}
					}
				}
			}
		}
	}

	return addrs
}

func queryTailscalePeers() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "tailscale", "status", "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var status struct {
		Peer map[string]struct {
			TailscaleIPs []string `json:"TailscaleIPs"`
			Online       bool     `json:"Online"`
		} `json:"Peer"`
	}
	if err := json.Unmarshal(out, &status); err != nil {
		return nil
	}

	var ips []string
	for _, p := range status.Peer {
		if !p.Online {
			continue
		}
		for _, ipStr := range p.TailscaleIPs {
			ip := net.ParseIP(ipStr)
			if ip != nil && ip.To4() != nil {
				ips = append(ips, ip.String())
				break
			}
		}
	}
	return ips
}

func (d *Discovery) listenLoop() {
	defer d.wg.Done()
	buf := make([]byte, 2048)

	for {
		select {
		case <-d.stop:
			return
		default:
		}

		if d.conn == nil {
			return
		}

		_ = d.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, remoteAddr, err := d.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		var packet BeaconPacket
		if err := json.Unmarshal(buf[:n], &packet); err != nil {
			continue
		}

		// Ignore beacons from self
		if packet.NodeID == "" || packet.NodeID == d.nodeID {
			continue
		}

		ip := remoteAddr.IP.String()
		port := packet.Port
		if port <= 0 {
			port = 3000
		}

		address := fmt.Sprintf("%s:%d", ip, port)

		d.mu.Lock()
		d.discovered[packet.NodeID] = &models.DiscoveredPeer{
			NodeID:   packet.NodeID,
			Name:     packet.Name,
			Address:  address,
			LastSeen: time.Now(),
		}
		d.mu.Unlock()
	}
}

// ListDiscovered returns all currently active discovered peers.
func (d *Discovery) ListDiscovered() []*models.DiscoveredPeer {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	var list []*models.DiscoveredPeer
	for id, peer := range d.discovered {
		if now.Sub(peer.LastSeen) > PeerExpiry {
			delete(d.discovered, id)
			continue
		}
		list = append(list, peer)
	}
	return list
}

// AddDiscovered manually registers a discovered peer (e.g. for testing or external hooks).
func (d *Discovery) AddDiscovered(p *models.DiscoveredPeer) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.discovered[p.NodeID] = p
}
