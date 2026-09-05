package models

import "time"

// TopologyNode represents a probing vantage point in the network graph.
type TopologyNode struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ZoneID    string    `json:"zone_id"`
	Scopes    []string  `json:"scopes"`
	Status    string    `json:"status"` // online, offline, degraded
	IsLocal   bool      `json:"is_local"`
	LastSeen  time.Time `json:"last_seen"`
}

// TopologyEdge represents a probe connection between a node and an endpoint.
type TopologyEdge struct {
	ID         string        `json:"id"`
	NodeID     string        `json:"node_id"`
	ServiceID  string        `json:"service_id"`
	EndpointID string        `json:"endpoint_id"`
	ScopeID    string        `json:"scope_id"`
	Status     MonitorStatus `json:"status"` // up, down, degraded
	LatencyMs  *int          `json:"latency_ms,omitempty"`
	Message    string        `json:"message,omitempty"`
	CheckedAt  time.Time     `json:"checked_at"`
}

// NetworkTopology represents the full visual graph of nodes, zones, services, and edges.
type NetworkTopology struct {
	Nodes    []*TopologyNode `json:"nodes"`
	Zones    []*Zone         `json:"zones"`
	Scopes   []*Scope        `json:"scopes"`
	Services []*Service      `json:"services"`
	Edges    []*TopologyEdge `json:"edges"`
}
