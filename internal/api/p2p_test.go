package api

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/p2p"
	"github.com/updu/updu/internal/storage"
)

func setupTestServerWithP2P(t *testing.T) (*Server, *storage.DB, *p2p.Manager, func()) {
	t.Helper()
	srv, db, cleanup := setupAPITest(t)

	ctx := context.Background()
	id, err := p2p.EnsureIdentity(ctx, db, "TestNode")
	if err != nil {
		t.Fatalf("p2p.EnsureIdentity failed: %v", err)
	}

	disc := p2p.NewDiscovery(id.NodeID, id.Name, 3000, 3001)
	p2pMgr := p2p.NewManager(id, disc, db, srv.sse, "127.0.0.1:3000")
	srv.SetP2P(p2pMgr)

	return srv, db, p2pMgr, cleanup
}

func TestP2PEndpoints(t *testing.T) {
	srv, _, p2pMgr, cleanup := setupTestServerWithP2P(t)
	defer cleanup()

	router := srv.Router()

	// 1. GET /api/v1/p2p/identity
	req := httptest.NewRequest("GET", "/api/v1/p2p/identity", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/p2p/identity returned status %d: %s", rec.Code, rec.Body.String())
	}

	var idResp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &idResp); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}
	if idResp["node_id"] == "" || idResp["public_key"] == "" {
		t.Fatalf("unexpected identity response: %+v", idResp)
	}

	// 2. POST /api/v1/p2p/pair-request
	remPub, remPriv, _ := ed25519.GenerateKey(rand.Reader)
	remNodeID := p2p.DeriveNodeID(remPub)
	remAddr := "192.168.1.55:3000"
	ts := time.Now().Unix()

	canonical := fmt.Sprintf("%s:%s:%s:%d", remNodeID, "RemotePeer", remAddr, ts)
	sig := fmt.Sprintf("%x", ed25519.Sign(remPriv, []byte(canonical)))
	pubHex := fmt.Sprintf("%x", remPub)

	proposal := models.PairProposal{
		NodeID:    remNodeID,
		Name:      "RemotePeer",
		Address:   remAddr,
		PublicKey: pubHex,
		Signature: sig,
		Timestamp: ts,
	}

	propData, _ := json.Marshal(proposal)
	reqPair := httptest.NewRequest("POST", "/api/v1/p2p/pair-request", bytes.NewReader(propData))
	reqPair.Header.Set("Content-Type", "application/json")
	recPair := httptest.NewRecorder()
	router.ServeHTTP(recPair, reqPair)

	if recPair.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/p2p/pair-request returned status %d: %s", recPair.Code, recPair.Body.String())
	}

	// 3. GET /api/v1/p2p/feed unauthorized without headers
	reqFeed := httptest.NewRequest("GET", "/api/v1/p2p/feed", nil)
	recFeed := httptest.NewRecorder()
	router.ServeHTTP(recFeed, reqFeed)

	if recFeed.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unsigned feed request, got %d", recFeed.Code)
	}

	// Approve the peer first so it can authenticate to /feed
	if err := p2pMgr.ApprovePeer(context.Background(), remNodeID); err != nil {
		t.Fatalf("ApprovePeer failed: %v", err)
	}

	// Now send signed request from approved peer
	feedTS := time.Now().Unix()
	feedMsg := fmt.Sprintf("%s:%d", remNodeID, feedTS)
	feedSig := fmt.Sprintf("%x", ed25519.Sign(remPriv, []byte(feedMsg)))

	reqAuthFeed := httptest.NewRequest("GET", "/api/v1/p2p/feed", nil)
	reqAuthFeed.Header.Set("X-Peer-ID", remNodeID)
	reqAuthFeed.Header.Set("X-Peer-Timestamp", fmt.Sprintf("%d", feedTS))
	reqAuthFeed.Header.Set("X-Peer-Signature", feedSig)
	recAuthFeed := httptest.NewRecorder()
	router.ServeHTTP(recAuthFeed, reqAuthFeed)

	if recAuthFeed.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/p2p/feed returned %d for authenticated peer: %s", recAuthFeed.Code, recAuthFeed.Body.String())
	}

	var syncResp p2p.SyncResponse
	if err := json.Unmarshal(recAuthFeed.Body.Bytes(), &syncResp); err != nil {
		t.Fatalf("failed to decode sync response: %v", err)
	}
	if syncResp.NodeID != idResp["node_id"] {
		t.Errorf("expected node ID %s, got %s", idResp["node_id"], syncResp.NodeID)
	}
	if syncResp.Telemetry == nil {
		t.Errorf("expected telemetry in sync response")
	}
}

func TestAdminPeerEndpoints(t *testing.T) {
	srv, db, _, cleanup := setupTestServerWithP2P(t)
	defer cleanup()

	adminCookie, _ := setupAdminAndViewer(t, srv)
	router := srv.Router()

	// 1. GET /api/v1/admin/peers
	req := httptest.NewRequest("GET", "/api/v1/admin/peers", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/admin/peers returned %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp["local"] == nil {
		t.Errorf("expected local identity in response")
	}

	// Create a pending peer
	ctx := context.Background()
	_ = db.CreatePeer(ctx, &models.Peer{
		ID:        "PEER-1234",
		Name:      "Pending Peer",
		Address:   "100.64.1.2:3000",
		PublicKey: "abcd1234abcd1234",
		Status:    models.PeerStatusPending,
	})

	// 2. POST /api/v1/admin/peers/approve
	body, _ := json.Marshal(map[string]string{"id": "PEER-1234"})
	reqApprove := httptest.NewRequest("POST", "/api/v1/admin/peers/approve", bytes.NewReader(body))
	reqApprove.Header.Set("Content-Type", "application/json")
	reqApprove.AddCookie(adminCookie)
	recApprove := httptest.NewRecorder()
	router.ServeHTTP(recApprove, reqApprove)

	if recApprove.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/admin/peers/approve returned %d: %s", recApprove.Code, recApprove.Body.String())
	}

	p, _ := db.GetPeer(ctx, "PEER-1234")
	if p.Status != models.PeerStatusApproved {
		t.Errorf("expected status approved, got %s", p.Status)
	}

	// 3. DELETE /api/v1/admin/peers/{id}
	reqDel := httptest.NewRequest("DELETE", "/api/v1/admin/peers/PEER-1234", nil)
	reqDel.AddCookie(adminCookie)
	recDel := httptest.NewRecorder()
	router.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Fatalf("DELETE /api/v1/admin/peers/PEER-1234 returned %d", recDel.Code)
	}

	pAfter, _ := db.GetPeer(ctx, "PEER-1234")
	if pAfter != nil {
		t.Errorf("expected peer to be deleted")
	}
}

