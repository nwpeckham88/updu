package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestTicketsAPI(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	srv.config.ForwardAuthEnabled = true
	srv.config.TrustedProxyCIDRs = []string{"192.0.2.0/24", "127.0.0.1/32"}

	router := srv.Router()
	adminCookie := registerAndLoginAdmin(t, router)

	// Create services:
	// svc_family (groups: ["family"])
	// svc_admin (groups: ["internal"])
	svcFamily := &models.Service{
		ID:      "svc_family",
		Name:    "Jellyfin Media",
		Type:    "web",
		Groups:  []string{"family"},
		Enabled: true,
	}
	svcAdmin := &models.Service{
		ID:      "svc_admin",
		Name:    "Proxmox Admin",
		Type:    "web",
		Groups:  []string{"internal"},
		Enabled: true,
	}
	if err := db.CreateService(context.Background(), svcFamily); err != nil {
		t.Fatalf("CreateService svcFamily failed: %v", err)
	}
	if err := db.CreateService(context.Background(), svcAdmin); err != nil {
		t.Fatalf("CreateService svcAdmin failed: %v", err)
	}

	// Helper for Alice (family group)
	aliceReq := func(method, path string, body []byte) *http.Request {
		r := httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Remote-User", "alice")
		r.Header.Set("Remote-Groups", "family")
		return r
	}

	// Helper for Bob (guest group)
	bobReq := func(method, path string, body []byte) *http.Request {
		r := httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Remote-User", "bob")
		r.Header.Set("Remote-Groups", "guest")
		return r
	}

	// 1. Alice creates ticket for svcFamily (allowed)
	body := map[string]any{
		"title":       "Video buffering",
		"description": "Episode 4 won't play smoothly",
		"severity":    "high",
		"service_id":  "svc_family",
	}
	b, _ := json.Marshal(body)
	req := aliceReq("POST", "/api/v1/tickets", b)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
	}
	var createdTicket models.Ticket
	if err := json.Unmarshal(rr.Body.Bytes(), &createdTicket); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if createdTicket.ServiceName != "Jellyfin Media" || createdTicket.CreatedBy != "alice" {
		t.Fatalf("unexpected ticket details: %+v", createdTicket)
	}

	// 2. Alice tries to create ticket for svcAdmin (forbidden: not in group)
	bodyAdmin := map[string]any{
		"title":       "Cannot access Proxmox",
		"description": "Port 8006 timeout",
		"service_id":  "svc_admin",
	}
	bAdmin, _ := json.Marshal(bodyAdmin)
	req2 := aliceReq("POST", "/api/v1/tickets", bAdmin)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d: %s", rr2.Code, rr2.Body.String())
	}

	// 3. Bob lists tickets -> sees 0 tickets (Alice's ticket is hidden)
	reqListBob := bobReq("GET", "/api/v1/tickets", nil)
	rrBob := httptest.NewRecorder()
	router.ServeHTTP(rrBob, reqListBob)
	if rrBob.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rrBob.Code)
	}
	var bobTickets []*models.Ticket
	json.Unmarshal(rrBob.Body.Bytes(), &bobTickets)
	if len(bobTickets) != 0 {
		t.Fatalf("expected 0 tickets for bob, got %d", len(bobTickets))
	}

	// 4. Bob tries to GET Alice's ticket directly -> 403 Forbidden
	reqGetBob := bobReq("GET", "/api/v1/tickets/"+createdTicket.ID, nil)
	rrGetBob := httptest.NewRecorder()
	router.ServeHTTP(rrGetBob, reqGetBob)
	if rrGetBob.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", rrGetBob.Code)
	}

	// 5. Admin lists tickets -> sees Alice's ticket
	reqListAdmin := httptest.NewRequest("GET", "/api/v1/tickets", nil)
	reqListAdmin.AddCookie(adminCookie)
	rrAdmin := httptest.NewRecorder()
	router.ServeHTTP(rrAdmin, reqListAdmin)
	if rrAdmin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rrAdmin.Code)
	}
	var adminTickets []*models.Ticket
	json.Unmarshal(rrAdmin.Body.Bytes(), &adminTickets)
	if len(adminTickets) != 1 || adminTickets[0].ID != createdTicket.ID {
		t.Fatalf("expected 1 ticket for admin, got %d", len(adminTickets))
	}

	// 6. Admin updates ticket status to in_progress
	updateBody := map[string]any{
		"status": "in_progress",
	}
	bUp, _ := json.Marshal(updateBody)
	reqUp := httptest.NewRequest("PUT", "/api/v1/tickets/"+createdTicket.ID, bytes.NewReader(bUp))
	reqUp.AddCookie(adminCookie)
	rrUp := httptest.NewRecorder()
	router.ServeHTTP(rrUp, reqUp)
	if rrUp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rrUp.Code, rrUp.Body.String())
	}
	var updatedTicket models.Ticket
	json.Unmarshal(rrUp.Body.Bytes(), &updatedTicket)
	if updatedTicket.Status != models.TicketStatusInProgress {
		t.Fatalf("expected in_progress, got %s", updatedTicket.Status)
	}

	// 7. Admin resolves ticket
	resolveBody := map[string]any{
		"status": "resolved",
	}
	bRes, _ := json.Marshal(resolveBody)
	reqRes := httptest.NewRequest("PUT", "/api/v1/tickets/"+createdTicket.ID, bytes.NewReader(bRes))
	reqRes.AddCookie(adminCookie)
	rrRes := httptest.NewRecorder()
	router.ServeHTTP(rrRes, reqRes)
	if rrRes.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rrRes.Code, rrRes.Body.String())
	}
	var resTicket models.Ticket
	json.Unmarshal(rrRes.Body.Bytes(), &resTicket)
	if resTicket.Status != models.TicketStatusResolved || resTicket.ResolvedAt == nil {
		t.Fatalf("expected resolved status with resolved_at, got %+v", resTicket)
	}

	// 8. Admin deletes ticket
	reqDel := httptest.NewRequest("DELETE", "/api/v1/tickets/"+createdTicket.ID, nil)
	reqDel.AddCookie(adminCookie)
	rrDel := httptest.NewRecorder()
	router.ServeHTTP(rrDel, reqDel)
	if rrDel.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content on delete, got %d", rrDel.Code)
	}
}

func TestMaintenanceViewerFiltering(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	srv.config.ForwardAuthEnabled = true
	srv.config.TrustedProxyCIDRs = []string{"192.0.2.0/24", "127.0.0.1/32"}

	router := srv.Router()
	adminCookie := registerAndLoginAdmin(t, router)

	ctx := context.Background()
	// Create monitors: mon_family (groups: ["family"]) and mon_admin (groups: ["internal"])
	monFamily := &models.Monitor{
		ID:        "mon_family",
		Name:      "Family Monitor",
		Type:      "http",
		Config:    json.RawMessage(`{"url":"http://127.0.0.1:8096"}`),
		Groups:    []string{"family"},
		IntervalS: 60,
		TimeoutS:  10,
		Enabled:   true,
		CreatedBy: "admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	monAdmin := &models.Monitor{
		ID:        "mon_admin",
		Name:      "Admin Monitor",
		Type:      "http",
		Config:    json.RawMessage(`{"url":"http://127.0.0.1:8006"}`),
		Groups:    []string{"internal"},
		IntervalS: 60,
		TimeoutS:  10,
		Enabled:   true,
		CreatedBy: "admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.CreateMonitor(ctx, monFamily); err != nil {
		t.Fatalf("CreateMonitor monFamily failed: %v", err)
	}
	if err := db.CreateMonitor(ctx, monAdmin); err != nil {
		t.Fatalf("CreateMonitor monAdmin failed: %v", err)
	}

	now := time.Now()
	mwFamily := &models.MaintenanceWindow{
		ID:         "mw_fam",
		Title:      "Family Plex Upgrade",
		MonitorIDs: []string{"mon_family"},
		StartsAt:   now.Add(-10 * time.Minute),
		EndsAt:     now.Add(1 * time.Hour),
		CreatedBy:  "admin",
	}
	mwAdmin := &models.MaintenanceWindow{
		ID:         "mw_adm",
		Title:      "Core Switch Firmware",
		MonitorIDs: []string{"mon_admin"},
		StartsAt:   now.Add(-10 * time.Minute),
		EndsAt:     now.Add(1 * time.Hour),
		CreatedBy:  "admin",
	}
	db.CreateMaintenanceWindow(ctx, mwFamily)
	db.CreateMaintenanceWindow(ctx, mwAdmin)

	// Alice queries maintenance
	reqAlice := httptest.NewRequest("GET", "/api/v1/maintenance", nil)
	reqAlice.Header.Set("Remote-User", "alice")
	reqAlice.Header.Set("Remote-Groups", "family")
	rrAlice := httptest.NewRecorder()
	router.ServeHTTP(rrAlice, reqAlice)

	if rrAlice.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rrAlice.Code)
	}
	var aliceWindows []*models.MaintenanceWindow
	json.Unmarshal(rrAlice.Body.Bytes(), &aliceWindows)
	if len(aliceWindows) != 1 || aliceWindows[0].ID != "mw_fam" {
		t.Fatalf("expected only mw_fam for alice, got %+v", aliceWindows)
	}

	// Admin queries maintenance
	reqAdmin := httptest.NewRequest("GET", "/api/v1/maintenance", nil)
	reqAdmin.AddCookie(adminCookie)
	rrAdmin := httptest.NewRecorder()
	router.ServeHTTP(rrAdmin, reqAdmin)

	if rrAdmin.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rrAdmin.Code)
	}
	var adminWindows []*models.MaintenanceWindow
	json.Unmarshal(rrAdmin.Body.Bytes(), &adminWindows)
	if len(adminWindows) != 2 {
		t.Fatalf("expected 2 maintenance windows for admin, got %d", len(adminWindows))
	}
}
