package models

import (
	"encoding/json"
	"time"
)

// PeerStatus constants
const (
	PeerStatusPending      = "pending"
	PeerStatusApproved     = "approved"
	PeerStatusRejected     = "rejected"
	PeerStatusDisconnected = "disconnected"
)

// PeerRole constants
const (
	PeerRolePeer  = "peer"
	PeerRoleAgent = "agent"
	PeerRoleAdmin = "admin"
)

// Peer represents a remote node federated with this updu instance.
type Peer struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Address   string          `json:"address"`
	PublicKey string          `json:"public_key"`
	Role      string          `json:"role"`
	Status    string          `json:"status"`
	LastSeen  *time.Time      `json:"last_seen,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at"`

	// Runtime/transient fields
	LatencyMs *int            `json:"latency_ms,omitempty"`
	Monitors  []*Monitor      `json:"monitors,omitempty"`
	Triage    *SurvivorTriage `json:"triage,omitempty"`
}

// PeerTelemetry carries host metrics reported by a peer.
type PeerTelemetry struct {
	TailnetIP string  `json:"tailnet_ip,omitempty"`
	PublicIP  string  `json:"public_ip,omitempty"`
	MemPct    float64 `json:"mem_pct"`
	CPUPct    float64 `json:"cpu_pct"`
	UptimeS   uint64  `json:"uptime_s"`
}

// TracerouteHop represents a single hop in a diagnostic hop trace.
type TracerouteHop struct {
	Hop     int     `json:"hop"`
	Address string  `json:"address"`
	RTTMs   float64 `json:"rtt_ms"`
	LossPct float64 `json:"loss_pct"`
}

// SurvivorTriage contains post-mortem triage data when a peer heartbeat fails.
type SurvivorTriage struct {
	PeerID             string          `json:"peer_id"`
	PeerName           string          `json:"peer_name"`
	DiscoveredAt       time.Time       `json:"discovered_at"`
	ProbableCause      string          `json:"probable_cause"`
	TailnetReachable   bool            `json:"tailnet_reachable"`
	PublicWANReachable bool            `json:"public_wan_reachable"`
	TailnetIP          string          `json:"tailnet_ip,omitempty"`
	PublicIP           string          `json:"public_ip,omitempty"`
	LastTelemetry      *PeerTelemetry  `json:"last_telemetry,omitempty"`
	TracerouteHops     []TracerouteHop `json:"traceroute_hops,omitempty"`
	Notes              string          `json:"notes,omitempty"`
}

// DiscoveredPeer represents a peer discovered via LAN UDP beacon.
type DiscoveredPeer struct {
	NodeID   string    `json:"node_id"`
	Name     string    `json:"name"`
	Address  string    `json:"address"`
	LastSeen time.Time `json:"last_seen"`
}

// PairProposal is the cryptographic pairing request sent between nodes.
type PairProposal struct {
	NodeID    string `json:"node_id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
	Timestamp int64  `json:"timestamp"`
}
