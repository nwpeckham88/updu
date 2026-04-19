package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/updu/updu/internal/updater"
)

func TestAPI_ApplyUpdate_SurfacesCheckError(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	adminCookie, _ := setupAdminAndViewer(t, srv)

	oldCheck := checkForUpdateForChannel
	defer func() { checkForUpdateForChannel = oldCheck }()
	checkForUpdateForChannel = func(string) (*updater.UpdateInfo, error) {
		return nil, errors.New("GitHub API rate limited (status 403)")
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/system/update", nil)
	req.AddCookie(adminCookie)
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if !strings.Contains(body["error"], "rate limited") {
		t.Fatalf("expected error to surface underlying cause, got %q", body["error"])
	}
}

func TestAPI_ApplyUpdate_SurfacesDownloadError(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	adminCookie, _ := setupAdminAndViewer(t, srv)

	oldCheck := checkForUpdateForChannel
	oldApply := downloadAndApplyUpdate
	oldRestart := scheduleRestart
	defer func() {
		checkForUpdateForChannel = oldCheck
		downloadAndApplyUpdate = oldApply
		scheduleRestart = oldRestart
	}()

	checkForUpdateForChannel = func(string) (*updater.UpdateInfo, error) {
		return &updater.UpdateInfo{
			CurrentVersion:  "v0.5.0",
			LatestVersion:   "v0.5.1",
			UpdateAvailable: true,
			AssetURL:        "https://example.invalid/updu",
			AssetName:       "updu",
		}, nil
	}
	downloadAndApplyUpdate = func(*updater.UpdateInfo) error {
		return errors.New("checksum mismatch: expected abc, got def")
	}

	var restartCalls int32
	scheduleRestart = func(time.Duration, string) { atomic.AddInt32(&restartCalls, 1) }

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/system/update", nil)
	req.AddCookie(adminCookie)
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if !strings.Contains(body["error"], "checksum mismatch") {
		t.Fatalf("expected error to surface checksum mismatch, got %q", body["error"])
	}
	if atomic.LoadInt32(&restartCalls) != 0 {
		t.Fatalf("restart must not be scheduled when download fails")
	}
}

func TestAPI_ApplyUpdate_SchedulesRestartOnSuccess(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	adminCookie, _ := setupAdminAndViewer(t, srv)

	oldCheck := checkForUpdateForChannel
	oldApply := downloadAndApplyUpdate
	oldRestart := scheduleRestart
	defer func() {
		checkForUpdateForChannel = oldCheck
		downloadAndApplyUpdate = oldApply
		scheduleRestart = oldRestart
	}()

	checkForUpdateForChannel = func(string) (*updater.UpdateInfo, error) {
		return &updater.UpdateInfo{
			CurrentVersion:  "v0.5.0",
			LatestVersion:   "v0.5.1",
			UpdateAvailable: true,
			AssetURL:        "https://example.invalid/updu",
			AssetName:       "updu",
		}, nil
	}
	downloadAndApplyUpdate = func(*updater.UpdateInfo) error { return nil }

	var (
		restartCalls  int32
		restartReason string
		restartDelay  time.Duration
	)
	scheduleRestart = func(d time.Duration, reason string) {
		atomic.AddInt32(&restartCalls, 1)
		restartDelay = d
		restartReason = reason
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/system/update", bytes.NewBuffer(nil))
	req.AddCookie(adminCookie)
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := atomic.LoadInt32(&restartCalls); got != 1 {
		t.Fatalf("expected 1 restart call, got %d", got)
	}
	if restartDelay <= 0 {
		t.Fatalf("expected positive restart delay, got %v", restartDelay)
	}
	if !strings.Contains(restartReason, "v0.5.1") {
		t.Fatalf("restart reason should reference target version, got %q", restartReason)
	}
}

func TestUpdater_ScheduleRestartIsIdempotent(t *testing.T) {
	// Use a long delay so the goroutine never fires during the test.
	updater.ScheduleRestart(1*time.Hour, "first")
	updater.ScheduleRestart(1*time.Hour, "second")

	if !updater.RestartRequested() {
		t.Fatal("expected RestartRequested() to be true after ScheduleRestart")
	}
	if updater.RestartReason() != "first" {
		t.Fatalf("expected reason to remain 'first' after second call, got %q", updater.RestartReason())
	}
}
