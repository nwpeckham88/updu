package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPI_Notifications(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// 1. Setup Admin User and get session cookie
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 2. Create Notification Channel
	ncReq, _ := json.Marshal(map[string]any{
		"name": "My Email", "type": "email", "config": map[string]string{"to": "test@example.com"},
	})
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBuffer(ncReq))
	req.AddCookie(sessionCookie)
	rrCreate := httptest.NewRecorder()
	router.ServeHTTP(rrCreate, req)

	if rrCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rrCreate.Code)
	}

	// Extract ID
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(rrCreate.Body).Decode(&created)

	// 3. List
	req = httptest.NewRequest("GET", "/api/v1/notifications", nil)
	req.AddCookie(sessionCookie)
	rrList := httptest.NewRecorder()
	router.ServeHTTP(rrList, req)

	if rrList.Code != http.StatusOK {
		t.Errorf("expected 200 OK for list")
	}

	// 4. Get by ID
	req = httptest.NewRequest("GET", "/api/v1/notifications/"+created.ID, nil)
	req.AddCookie(sessionCookie)
	rrGet := httptest.NewRecorder()
	router.ServeHTTP(rrGet, req)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected 200 OK for get")
	}

	// 5. Update
	upReq, _ := json.Marshal(map[string]any{
		"name": "Updated Email", "type": "email", "config": map[string]string{"to": "updated@example.com"},
	})
	req = httptest.NewRequest("PUT", "/api/v1/notifications/"+created.ID, bytes.NewBuffer(upReq))
	req.AddCookie(sessionCookie)
	rrUpdate := httptest.NewRecorder()
	router.ServeHTTP(rrUpdate, req)
	if rrUpdate.Code != http.StatusOK {
		t.Errorf("expected 200 OK for update")
	}

	// 6. Delete
	req = httptest.NewRequest("DELETE", "/api/v1/notifications/"+created.ID, nil)
	req.AddCookie(sessionCookie)
	rrDelete := httptest.NewRecorder()
	router.ServeHTTP(rrDelete, req)
	if rrDelete.Code != http.StatusOK {
		t.Errorf("expected 200 OK for delete")
	}
}
