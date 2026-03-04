package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPI_AdminGaps(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// 1. Setup Admin User
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 2. Groups API
	// Create a monitor with a group
	monBody, _ := json.Marshal(map[string]any{
		"name": "M1", "type": "http", "group_name": "Group A", "config": map[string]string{"url": "http://a.com"},
	})
	req := httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer(monBody))
	req.AddCookie(sessionCookie)
	rrMon := httptest.NewRecorder()
	router.ServeHTTP(rrMon, req)

	var mon struct {
		ID string `json:"id"`
	}
	json.NewDecoder(rrMon.Body).Decode(&mon)

	// List Groups
	req = httptest.NewRequest("GET", "/api/v1/groups", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for list groups, got %d", rr.Code)
	}

	// Update Group
	updateGrpBody, _ := json.Marshal(map[string]string{"name": "Group B"})
	req = httptest.NewRequest("PUT", "/api/v1/groups/Group%20A", bytes.NewBuffer(updateGrpBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for update group, got %d", rr.Code)
	}

	// Delete Group
	req = httptest.NewRequest("DELETE", "/api/v1/groups/Group%20B", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for delete group, got %d", rr.Code)
	}

	// 3. User Management API
	// List Users
	req = httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for list users, got %d", rr.Code)
	}

	// Create another user for testing (requires admin cookie after first user)
	u2Body, _ := json.Marshal(map[string]string{"username": "viewer", "password": "password123"})
	rrReg := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(u2Body))
	req2.AddCookie(sessionCookie)
	router.ServeHTTP(rrReg, req2)

	if rrReg.Code != http.StatusOK {
		t.Fatalf("expected 200 for second user register, got %d: %s", rrReg.Code, rrReg.Body.String())
	}

	var u2 struct {
		ID string `json:"id"`
	}
	json.NewDecoder(rrReg.Body).Decode(&u2)
	if u2.ID == "" {
		t.Fatal("expected user ID in registration response, got empty")
	}

	// Update User Role
	roleBody, _ := json.Marshal(map[string]string{"role": "admin"})
	req = httptest.NewRequest("PUT", "/api/v1/admin/users/"+u2.ID+"/role", bytes.NewBuffer(roleBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for update role, got %d: %s", rr.Code, rr.Body.String())
	}

	// Delete User
	req = httptest.NewRequest("DELETE", "/api/v1/admin/users/"+u2.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for delete user, got %d", rr.Code)
	}

	// 4. Settings API
	// Update Settings
	settingsBody, _ := json.Marshal(map[string]string{"site_name": "Updu Pro"})
	req = httptest.NewRequest("POST", "/api/v1/settings", bytes.NewBuffer(settingsBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for update settings, got %d", rr.Code)
	}

	// Get Settings
	req = httptest.NewRequest("GET", "/api/v1/settings", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for get settings, got %d", rr.Code)
	}

	// 5. Incidents API
	incBody, _ := json.Marshal(map[string]any{
		"monitor_id": mon.ID, "title": "Test Incident", "status": "investigating", "description": "Uh oh",
	})
	req = httptest.NewRequest("POST", "/api/v1/incidents", bytes.NewBuffer(incBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create incident, got %d", rr.Code)
	}
	var inc struct {
		ID string `json:"id"`
	}
	json.NewDecoder(rr.Body).Decode(&inc)

	// List Incidents
	req = httptest.NewRequest("GET", "/api/v1/incidents", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for list incidents, got %d", rr.Code)
	}

	// Get Incident
	req = httptest.NewRequest("GET", "/api/v1/incidents/"+inc.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for get incident, got %d", rr.Code)
	}

	// Update Incident
	incUpBody, _ := json.Marshal(map[string]any{
		"status": "resolved", "description": "All good",
	})
	req = httptest.NewRequest("PUT", "/api/v1/incidents/"+inc.ID, bytes.NewBuffer(incUpBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for update incident, got %d", rr.Code)
	}

	// Delete Incident
	req = httptest.NewRequest("DELETE", "/api/v1/incidents/"+inc.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for delete incident, got %d", rr.Code)
	}

	// 6. Maintenance API
	maintBody, _ := json.Marshal(map[string]any{
		"title": "DB Maintenance", "start_time": "2024-01-01T00:00:00Z", "end_time": "2024-01-01T01:00:00Z", "monitors": []string{mon.ID},
	})
	req = httptest.NewRequest("POST", "/api/v1/maintenance", bytes.NewBuffer(maintBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create maintenance, got %d", rr.Code)
	}
	var maint struct {
		ID string `json:"id"`
	}
	json.NewDecoder(rr.Body).Decode(&maint)

	// List Maintenance Windows
	req = httptest.NewRequest("GET", "/api/v1/maintenance", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for list maintenance, got %d", rr.Code)
	}

	// Get Maintenance Window
	req = httptest.NewRequest("GET", "/api/v1/maintenance/"+maint.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for get maintenance, got %d", rr.Code)
	}

	// Update Maintenance Window
	maintUpBody, _ := json.Marshal(map[string]any{
		"title": "System Maintenance", "start_time": "2024-01-01T00:00:00Z", "end_time": "2024-01-01T02:00:00Z", "monitors": []string{mon.ID},
	})
	req = httptest.NewRequest("PUT", "/api/v1/maintenance/"+maint.ID, bytes.NewBuffer(maintUpBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for update maintenance, got %d", rr.Code)
	}

	// Delete Maintenance Window
	req = httptest.NewRequest("DELETE", "/api/v1/maintenance/"+maint.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for delete maintenance, got %d", rr.Code)
	}
}
