package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/updu/updu/internal/checker"
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

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	ctx := context.WithValue(context.Background(), checker.AllowLocalhostKey, true)

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
				Config:     json.RawMessage(`{"url":"` + ts.URL + `"}`),
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
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"list_zones","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"probe_service","arguments":{"service_id":"srv-vaultwarden"}}}`,
	}, "\n") + "\n"

	reader := strings.NewReader(input)
	var output bytes.Buffer

	server := mcp.NewServer(db, reader, &output)
	if err := server.Run(ctx); err != nil {
		t.Fatalf("server.Run returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 7 {
		t.Fatalf("expected 7 JSON-RPC responses, got %d. Output: %s", len(lines), output.String())
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
	toolsList := resMap["tools"].([]any)
	foundProbe := false
	foundZones := false
	for _, t := range toolsList {
		tm := t.(map[string]any)
		if tm["name"] == "probe_service" {
			foundProbe = true
		}
		if tm["name"] == "list_zones" {
			foundZones = true
		}
	}
	if !foundProbe {
		t.Errorf("expected tools/list to include probe_service")
	}
	if !foundZones {
		t.Errorf("expected tools/list to include list_zones")
	}

	// Verify list_services response contains Vaultwarden
	if !strings.Contains(lines[2], "Vaultwarden") {
		t.Errorf("expected list_services output to contain Vaultwarden: %s", lines[2])
	}

	// Verify list_zones response
	if !strings.Contains(lines[5], "default") {
		t.Errorf("expected list_zones output to contain default zone: %s", lines[5])
	}

	// Verify probe_service response executed probe against test server
	if !strings.Contains(lines[6], "srv-vaultwarden") || !strings.Contains(lines[6], "healthy") {
		t.Errorf("expected probe_service output to contain healthy srv-vaultwarden: %s", lines[6])
	}
}
