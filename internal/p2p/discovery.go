package p2p

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
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

// Discovery manages UDP beacon broadcasting and listening for LAN discovery.
type Discovery struct {
	nodeID   string
	name     string
	httpPort int
	udpPort  int

	conn *net.UDPConn
	stop chan struct{}
	wg   sync.WaitGroup

	mu         sync.RWMutex
	discovered map[string]*models.DiscoveredPeer
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

	broadcastAddr := &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: d.udpPort,
	}

	// Create an outbound UDP client with broadcast enabled
	outConn, err := net.DialUDP("udp4", nil, broadcastAddr)
	if err != nil {
		return
	}
	defer outConn.Close()

	_, _ = outConn.Write(data)
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
