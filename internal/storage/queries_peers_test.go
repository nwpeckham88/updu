package storage

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestPeerQueries(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_peers.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	peer := &models.Peer{
		ID:        "7X9K-3B21",
		Name:      "Node Alpha",
		Address:   "192.168.1.100:3000",
		PublicKey: "deadbeef12345678",
		Role:      models.PeerRolePeer,
		Status:    models.PeerStatusPending,
		Metadata:  json.RawMessage(`{"tailnet_ip":"100.64.0.1"}`),
	}

	if err := db.CreatePeer(ctx, peer); err != nil {
		t.Fatalf("CreatePeer failed: %v", err)
	}

	fetched, err := db.GetPeer(ctx, "7X9K-3B21")
	if err != nil {
		t.Fatalf("GetPeer failed: %v", err)
	}
	if fetched == nil || fetched.Name != "Node Alpha" {
		t.Fatalf("unexpected fetched peer: %+v", fetched)
	}

	byPub, err := db.GetPeerByPublicKey(ctx, "deadbeef12345678")
	if err != nil || byPub == nil || byPub.ID != "7X9K-3B21" {
		t.Fatalf("GetPeerByPublicKey failed: %v, %+v", err, byPub)
	}

	// Update status to approved
	if err := db.UpdatePeerStatus(ctx, "7X9K-3B21", models.PeerStatusApproved); err != nil {
		t.Fatalf("UpdatePeerStatus failed: %v", err)
	}

	approved, err := db.ListApprovedPeers(ctx)
	if err != nil || len(approved) != 1 {
		t.Fatalf("ListApprovedPeers failed: %v, len=%d", err, len(approved))
	}

	// Update last seen
	now := time.Now().Truncate(time.Second)
	if err := db.UpdatePeerLastSeen(ctx, "7X9K-3B21", now, json.RawMessage(`{"tailnet_ip":"100.64.0.2"}`)); err != nil {
		t.Fatalf("UpdatePeerLastSeen failed: %v", err)
	}

	fetched2, _ := db.GetPeer(ctx, "7X9K-3B21")
	if fetched2.LastSeen == nil {
		t.Fatalf("expected last seen to be set")
	}

	// Delete peer
	if err := db.DeletePeer(ctx, "7X9K-3B21"); err != nil {
		t.Fatalf("DeletePeer failed: %v", err)
	}

	afterDel, err := db.GetPeer(ctx, "7X9K-3B21")
	if err != nil || afterDel != nil {
		t.Fatalf("expected nil after delete, got %v, %v", afterDel, err)
	}
}
