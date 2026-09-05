package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/updu/updu/internal/mcp"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/storage"
)

func setupTestDB(t *testing.T) (*storage.DB, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "updu-mcp-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("opening test db: %v", err)
	}

	if err := db.Migrate(context.Background()); err != nil {
		db.Close()
		os.RemoveAll(dir)
		t.Fatalf("migrating test db: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}
	return db, cleanup
}

func TestMCPServer(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Seed a service
	svc := &models.Service{
		ID:     "srv-vaultwarden",
		Name:   "Vaultwarden",
		Type:   models.ServiceTypeWeb,
		ZoneID: "default",
		Endpoints: []*models.ServiceEndpoint{
			{
				ID:         "ep-wan",
				Name:       "Public Domain",
				ScopeID:    models.ScopePublic,
				TargetType: "http",
				Config:     json.RawMessage(`{"url":"https://example.com"}`),
				IsPrimary:  true,
			},
		},
	}
	if err := db.CreateService(ctx, svc); err != nil {
		t.Fatalf("CreateService failed: %v", err)
	}

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_services","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_service_details","arguments":{"service_id":"srv-vaultwarden"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_network_topology","arguments":{}}}`,
	}, "\n") + "\n"

	reader := strings.NewReader(input)
	var output bytes.Buffer

	server := mcp.NewServer(db, reader, &output)
	if err := server.Run(ctx); err != nil {
		t.Fatalf("server.Run returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 JSON-RPC responses, got %d. Output: %s", len(lines), output.String())
	}

	// Verify initialize response
	var initResp mcp.JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("unmarshaling init resp: %v", err)
	}
	if initResp.ID != float64(1) || initResp.Error != nil {
		t.Errorf("init response failed: %+v", initResp)
	}

	// Verify tools/list response
	var toolsResp mcp.JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[1]), &toolsResp); err != nil {
		t.Fatalf("unmarshaling tools resp: %v", err)
	}
	resMap, ok := toolsResp.Result.(map[string]any)
	if !ok || resMap["tools"] == nil {
		t.Fatalf("invalid tools/list result: %+v", toolsResp)
	}

	// Verify list_services response contains Vaultwarden
	if !strings.Contains(lines[2], "Vaultwarden") {
		t.Errorf("expected list_services output to contain Vaultwarden: %s", lines[2])
	}
}
