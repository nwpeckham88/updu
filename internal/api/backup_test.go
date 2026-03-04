package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPI_ConfigBackup(t *testing.T) {
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

	// 2. Create some data to backup
	// Monitor
	monReq, _ := json.Marshal(map[string]any{
		"name": "Backup Monitor", "type": "http", "config": map[string]string{"url": "http://example.com"},
	})
	req := httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer(monReq))
	req.AddCookie(sessionCookie)
	router.ServeHTTP(httptest.NewRecorder(), req)

	// Setting
	setReq, _ := json.Marshal(map[string]string{"test_setting": "test_value"})
	req = httptest.NewRequest("POST", "/api/v1/settings", bytes.NewBuffer(setReq))
	req.AddCookie(sessionCookie)
	router.ServeHTTP(httptest.NewRecorder(), req)

	// 3. Export Data
	req = httptest.NewRequest("GET", "/api/v1/system/backup", nil)
	req.AddCookie(sessionCookie)
	rrExport := httptest.NewRecorder()
	router.ServeHTTP(rrExport, req)

	if rrExport.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d for export", rrExport.Code)
	}

	var backup BackupConfig
	if err := json.NewDecoder(rrExport.Body).Decode(&backup); err != nil {
		t.Fatalf("failed decoding backup: %v", err)
	}

	if len(backup.Monitors) != 1 {
		t.Errorf("expected 1 monitor in backup, got %d", len(backup.Monitors))
	}
	if backup.Settings["test_setting"] != "test_value" {
		t.Errorf("expected setting 'test_value', got %v", backup.Settings["test_setting"])
	}

	// 4. Import Data
	importBody, _ := json.Marshal(backup)
	req = httptest.NewRequest("POST", "/api/v1/system/backup", bytes.NewBuffer(importBody))
	req.AddCookie(sessionCookie)
	rrImport := httptest.NewRecorder()
	router.ServeHTTP(rrImport, req)

	if rrImport.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d for import", rrImport.Code)
	}
}
