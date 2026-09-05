package p2p

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/storage"
)

func TestNodeIdentity(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "p2p_test.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("storage.Open failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	id1, err := EnsureIdentity(ctx, db, "TestNode1")
	if err != nil {
		t.Fatalf("EnsureIdentity failed: %v", err)
	}

	if len(id1.NodeID) != 9 || !strings.Contains(id1.NodeID, "-") {
		t.Fatalf("expected node ID format XXXX-XXXX, got %s", id1.NodeID)
	}

	// Signature verification
	data := []byte("hello-updu-p2p")
	sig := id1.Sign(data)
	if !VerifySignature(id1.PubKeyHex, data, sig) {
		t.Fatalf("VerifySignature failed for valid signature")
	}

	if VerifySignature(id1.PubKeyHex, []byte("tampered-data"), sig) {
		t.Fatalf("VerifySignature passed for tampered data")
	}

	// Reload identity from DB
	id2, err := EnsureIdentity(ctx, db, "")
	if err != nil {
		t.Fatalf("reload EnsureIdentity failed: %v", err)
	}
	if id2.NodeID != id1.NodeID || id2.PubKeyHex != id1.PubKeyHex {
		t.Fatalf("expected reloaded identity to match: %s != %s", id2.NodeID, id1.NodeID)
	}
}

func TestTelemetryCollection(t *testing.T) {
	telem := CollectTelemetry()
	if telem == nil {
		t.Fatalf("expected telemetry, got nil")
	}
	// MemPct should be >= 0 and <= 100
	if telem.MemPct < 0 || telem.MemPct > 100 {
		t.Errorf("invalid MemPct: %f", telem.MemPct)
	}
	// CPUPct should be >= 0 and <= 100
	if telem.CPUPct < 0 || telem.CPUPct > 100 {
		t.Errorf("invalid CPUPct: %f", telem.CPUPct)
	}
}

func TestSurvivorTriage(t *testing.T) {
	ctx := context.Background()

	peer := &models.Peer{
		ID:      "TEST-NODE",
		Name:    "Remote Machine",
		Address: "127.0.0.1:54321", // Non-existent port
	}

	// Test with normal telemetry
	telem := &models.PeerTelemetry{
		TailnetIP: "100.64.0.99",
		PublicIP:  "198.51.100.1",
		MemPct:    45.0,
		CPUPct:    20.0,
		UptimeS:   3600,
	}

	triage := RunTriage(ctx, peer, telem)
	if triage == nil {
		t.Fatalf("expected triage, got nil")
	}
	if triage.ProbableCause == "" {
		t.Errorf("expected probable cause to be set")
	}
	if len(triage.TracerouteHops) == 0 {
		t.Errorf("expected traceroute hops to be recorded")
	}

	// Test OOM kill diagnosis
	telemOOM := &models.PeerTelemetry{
		TailnetIP: "100.64.0.99",
		PublicIP:  "198.51.100.1",
		MemPct:    94.5,
		CPUPct:    30.0,
		UptimeS:   3600,
	}
	triageOOM := RunTriage(ctx, peer, telemOOM)
	if !strings.Contains(triageOOM.ProbableCause, "OOM") && !strings.Contains(triageOOM.Notes, "OOM") {
		t.Errorf("expected OOM notes or cause, got %s: %s", triageOOM.ProbableCause, triageOOM.Notes)
	}
}

func TestPairProposalValidation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "p2p_pair_test.db")
	db, _ := storage.Open(dbPath)
	defer db.Close()
	ctx := context.Background()
	_ = db.Migrate(ctx)

	id, _ := EnsureIdentity(ctx, db, "LocalNode")
	mgr := NewManager(id, nil, db, nil, "127.0.0.1:3000")

	// Generate remote node keypair
	remPub, remPriv, _ := ed25519.GenerateKey(rand.Reader)
	remNodeID := DeriveNodeID(remPub)
	remAddr := "192.168.1.50:3000"
	ts := time.Now().Unix()

	canonical := fmt.Sprintf("%s:%s:%s:%d", remNodeID, "RemoteNode", remAddr, ts)
	sig := fmt.Sprintf("%x", ed25519.Sign(remPriv, []byte(canonical)))
	pubHex := fmt.Sprintf("%x", remPub)

	proposal := &models.PairProposal{
		NodeID:    remNodeID,
		Name:      "RemoteNode",
		Address:   remAddr,
		PublicKey: pubHex,
		Signature: sig,
		Timestamp: ts,
	}

	err := mgr.HandlePairProposal(ctx, proposal)
	if err != nil {
		t.Fatalf("HandlePairProposal failed: %v", err)
	}

	// Peer should be in pending state
	p, err := db.GetPeer(ctx, remNodeID)
	if err != nil || p == nil {
		t.Fatalf("expected stored peer, got err: %v", err)
	}
	if p.Status != models.PeerStatusPending {
		t.Errorf("expected status pending, got %s", p.Status)
	}

	// Approve peer
	if err := mgr.ApprovePeer(ctx, remNodeID); err != nil {
		t.Fatalf("ApprovePeer failed: %v", err)
	}

	pApproved, _ := db.GetPeer(ctx, remNodeID)
	if pApproved.Status != models.PeerStatusApproved {
		t.Errorf("expected status approved, got %s", pApproved.Status)
	}
}

func TestDiscoveryCache(t *testing.T) {
	disc := NewDiscovery("NODE-1234", "MyNode", 3000, 3001)
	disc.AddDiscovered(&models.DiscoveredPeer{
		NodeID:   "NODE-5678",
		Name:     "DiscoveredNode",
		Address:  "192.168.1.80:3000",
		LastSeen: time.Now(),
	})

	list := disc.ListDiscovered()
	if len(list) != 1 || list[0].NodeID != "NODE-5678" {
		t.Fatalf("unexpected discovered list: %+v", list)
	}
}

func TestAddressResolution(t *testing.T) {
	// Specific host given
	addr := ResolveAdvertisedAddress("192.168.1.100", 3000)
	if addr != "192.168.1.100:3000" {
		t.Errorf("expected 192.168.1.100:3000, got %s", addr)
	}

	// Environment variable override
	t.Setenv("UPDU_ADVERTISED_ADDRESS", "http://100.64.0.1:3000")
	addr = ResolveAdvertisedAddress("0.0.0.0", 3000)
	if addr != "100.64.0.1:3000" {
		t.Errorf("expected 100.64.0.1:3000, got %s", addr)
	}
}

func TestIsInternalAddr(t *testing.T) {
	cases := []struct {
		addr     string
		expected bool
	}{
		{"100.64.0.1:3000", true},
		{"http://100.64.0.4:3000", true},
		{"https://100.64.0.4:3000", true},
		{"192.168.2.4:3000", true},
		{"10.0.0.5:3000", true},
		{"127.0.0.1:3000", true},
		{"localhost:3000", true},
		{"my-host.tailnet.kn8design.com", true},
		{"updu.kn8design.com", false},
		{"https://updu.kn8design.com", false},
		{"8.8.8.8:3000", false},
	}

	for _, tc := range cases {
		actual := isInternalAddr(tc.addr)
		if actual != tc.expected {
			t.Errorf("isInternalAddr(%q) = %v, expected %v", tc.addr, actual, tc.expected)
		}
	}
}
