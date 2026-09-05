package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/p2p"
)

// SetP2P attaches a P2P manager to the API server.
func (s *Server) SetP2P(m *p2p.Manager) {
	s.p2p = m
}

// P2P returns the attached P2P manager.
func (s *Server) P2P() *p2p.Manager {
	return s.p2p
}

// GET /api/v1/p2p/identity
func (s *Server) handleP2PIdentity(w http.ResponseWriter, r *http.Request) {
	if s.p2p == nil {
		jsonError(w, "p2p not enabled", http.StatusServiceUnavailable)
		return
	}
	id := s.p2p.Identity()
	jsonOK(w, map[string]string{
		"node_id":    id.NodeID,
		"name":       id.Name,
		"public_key": id.PubKeyHex,
	})
}

// POST /api/v1/p2p/pair-request
func (s *Server) handleP2PPairRequest(w http.ResponseWriter, r *http.Request) {
	if s.p2p == nil {
		jsonError(w, "p2p not enabled", http.StatusServiceUnavailable)
		return
	}

	var prop models.PairProposal
	if err := json.NewDecoder(r.Body).Decode(&prop); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.p2p.HandlePairProposal(r.Context(), &prop); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonOK(w, map[string]string{
		"status":  "pending",
		"message": "pairing proposal received",
	})
}

// GET /api/v1/p2p/feed
func (s *Server) handleP2PFeed(w http.ResponseWriter, r *http.Request) {
	if s.p2p == nil {
		jsonError(w, "p2p not enabled", http.StatusServiceUnavailable)
		return
	}

	peerID := r.Header.Get("X-Peer-ID")
	ts := r.Header.Get("X-Peer-Timestamp")
	sig := r.Header.Get("X-Peer-Signature")

	if peerID == "" || ts == "" || sig == "" {
		jsonError(w, "missing peer authentication headers", http.StatusUnauthorized)
		return
	}

	if !s.p2p.AuthenticatePeer(r.Context(), peerID, ts, sig) {
		jsonError(w, "unauthorized peer", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	monitors, err := s.db.ListMonitors(ctx)
	if err != nil {
		jsonError(w, "failed to list monitors", http.StatusInternalServerError)
		return
	}

	// Enrich monitor models with runtime state from check_results
	for _, m := range monitors {
		var st string
		var lat sql.NullInt64
		var at time.Time
		if err := s.db.QueryRowContext(ctx, `
			SELECT status, latency_ms, checked_at FROM check_results
			WHERE monitor_id = ? ORDER BY checked_at DESC LIMIT 1
		`, m.ID).Scan(&st, &lat, &at); err == nil {
			m.Status = models.MonitorStatus(st)
			m.LastCheck = &at
			if lat.Valid {
				l := int(lat.Int64)
				m.LastLatency = &l
			}
		}
	}

	telem := p2p.CollectTelemetry()
	id := s.p2p.Identity()

	resp := p2p.SyncResponse{
		NodeID:    id.NodeID,
		Name:      id.Name,
		Telemetry: telem,
		Monitors:  monitors,
		Timestamp: time.Now().Unix(),
	}

	jsonOK(w, resp)
}

// GET /api/v1/admin/peers
func (s *Server) handleListPeers(w http.ResponseWriter, r *http.Request) {
	if s.p2p == nil {
		jsonOK(w, map[string]any{
			"peers":      []any{},
			"discovered": []any{},
			"triage":     []any{},
		})
		return
	}

	ctx := r.Context()
	peers, err := s.p2p.ListPeers(ctx)
	if err != nil {
		jsonError(w, "failed to list peers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var discovered []*models.DiscoveredPeer
	if disc := s.p2p.Discovery(); disc != nil {
		discovered = disc.ListDiscovered()
	}

	triage := s.p2p.ListActiveTriage()
	id := s.p2p.Identity()

	jsonOK(w, map[string]any{
		"local": map[string]string{
			"node_id":    id.NodeID,
			"name":       id.Name,
			"public_key": id.PubKeyHex,
		},
		"peers":      peers,
		"discovered": discovered,
		"triage":     triage,
	})
}

type approvePeerRequest struct {
	ID string `json:"id"`
}

// POST /api/v1/admin/peers/approve
func (s *Server) handleApprovePeer(w http.ResponseWriter, r *http.Request) {
	if s.p2p == nil {
		jsonError(w, "p2p not enabled", http.StatusServiceUnavailable)
		return
	}

	var req approvePeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		jsonError(w, "invalid request body: id required", http.StatusBadRequest)
		return
	}

	if err := s.p2p.ApprovePeer(r.Context(), req.ID); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonOK(w, map[string]string{"status": "approved"})
}

type connectPeerRequest struct {
	Address string `json:"address"`
	Name    string `json:"name,omitempty"`
}

// POST /api/v1/admin/peers/connect
func (s *Server) handleConnectPeer(w http.ResponseWriter, r *http.Request) {
	if s.p2p == nil {
		jsonError(w, "p2p not enabled", http.StatusServiceUnavailable)
		return
	}

	var req connectPeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Address == "" {
		jsonError(w, "invalid request body: address required", http.StatusBadRequest)
		return
	}

	peer, err := s.p2p.ConnectPeer(r.Context(), req.Address, req.Name)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonOK(w, peer)
}

// DELETE /api/v1/admin/peers/{id}
func (s *Server) handleDeletePeer(w http.ResponseWriter, r *http.Request) {
	if s.p2p == nil {
		jsonError(w, "p2p not enabled", http.StatusServiceUnavailable)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		jsonError(w, "missing peer id", http.StatusBadRequest)
		return
	}

	if err := s.p2p.RejectPeer(r.Context(), id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"status": "deleted"})
}
