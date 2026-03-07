package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	tempDir, err := os.MkdirTemp("", "updu-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return db, cleanup
}

func TestUserQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	user := &models.User{
		ID:        "user-1",
		Username:  "testuser",
		Password:  "hashedpassword",
		Role:      models.RoleAdmin,
		CreatedAt: time.Now(),
	}

	if err := db.CreateUser(ctx, user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	got, err := db.GetUserByUsername(ctx, "testuser")
	if err != nil || got == nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, got.ID)
	}

	count, _ := db.CountUsers(ctx)
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}

func TestMonitorQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	m := &models.Monitor{
		ID:        "mon-1",
		Name:      "Test Monitor",
		Type:      "http",
		Config:    []byte(`{"url":"http://example.com"}`),
		IntervalS: 60,
		Enabled:   true,
		CreatedBy: "user-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.CreateMonitor(ctx, m); err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	got, err := db.GetMonitor(ctx, m.ID)
	if err != nil || got == nil {
		t.Fatalf("failed to get monitor: %v", err)
	}
	if got.Name != m.Name {
		t.Errorf("expected name %s, got %s", m.Name, got.Name)
	}

	monitors, _ := db.ListMonitors(ctx)
	if len(monitors) != 1 {
		t.Errorf("expected 1 monitor, got %d", len(monitors))
	}

	m.Name = "Updated Name"
	if err := db.UpdateMonitor(ctx, m); err != nil {
		t.Fatalf("failed to update monitor: %v", err)
	}

	if err := db.DeleteMonitor(ctx, m.ID); err != nil {
		t.Fatalf("failed to delete monitor: %v", err)
	}
}

func TestIncidentQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	i := &models.Incident{
		ID:         "inc-1",
		Title:      "Test Incident",
		Status:     models.IncidentInvestigating,
		Severity:   "minor",
		MonitorIDs: []string{"mon-1"},
		StartedAt:  time.Now(),
		CreatedBy:  "user-1",
	}

	if err := db.CreateIncident(ctx, i); err != nil {
		t.Fatalf("failed to create incident: %v", err)
	}

	incidents, _ := db.ListIncidents(ctx)
	if len(incidents) != 1 {
		t.Errorf("expected 1 incident, got %d", len(incidents))
	}

	i.Status = models.IncidentResolved
	now := time.Now()
	i.ResolvedAt = &now
	if err := db.UpdateIncident(ctx, i); err != nil {
		t.Fatalf("failed to update incident: %v", err)
	}

	if err := db.DeleteIncident(ctx, i.ID); err != nil {
		t.Fatalf("failed to delete incident: %v", err)
	}
}

func TestMaintenanceQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	mw := &models.MaintenanceWindow{
		ID:         "mw-1",
		Title:      "Maintenance",
		MonitorIDs: []string{"mon-1"},
		StartsAt:   time.Now(),
		EndsAt:     time.Now().Add(time.Hour),
		CreatedBy:  "user-1",
	}

	if err := db.CreateMaintenanceWindow(ctx, mw); err != nil {
		t.Fatalf("failed to create maintenance window: %v", err)
	}

	windows, _ := db.ListMaintenanceWindows(ctx)
	if len(windows) != 1 {
		t.Errorf("expected 1 window, got %d", len(windows))
	}

	if err := db.DeleteMaintenanceWindow(ctx, mw.ID); err != nil {
		t.Fatalf("failed to delete window: %v", err)
	}

	// 2. Update Maintenance Window
	mw2 := &models.MaintenanceWindow{
		ID:         "mw-2",
		Title:      "Maintenance 2",
		MonitorIDs: []string{"mon-1"},
		StartsAt:   time.Now(),
		EndsAt:     time.Now().Add(time.Hour),
		CreatedBy:  "user-1",
	}
	db.CreateMaintenanceWindow(ctx, mw2)
	mw2.Title = "Updated Maintenance"
	if err := db.UpdateMaintenanceWindow(ctx, mw2); err != nil {
		t.Fatalf("failed to update maintenance window: %v", err)
	}
	gotMW, _ := db.GetMaintenanceWindow(ctx, mw2.ID)
	if gotMW.Title != "Updated Maintenance" {
		t.Errorf("expected Updated Maintenance, got %s", gotMW.Title)
	}

	// 3. IsMonitorUnderMaintenance
	active, _ := db.IsMonitorUnderMaintenance(ctx, "mon-1")
	if !active {
		t.Errorf("expected monitor to be under maintenance")
	}
	active, _ = db.IsMonitorUnderMaintenance(ctx, "mon-2")
	if active {
		t.Errorf("expected monitor mon-2 NOT to be under maintenance")
	}
}

func TestNotificationChannelQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	nc := &models.NotificationChannel{
		ID:      "nc-1",
		Name:    "Slack",
		Type:    "slack",
		Config:  map[string]any{"url": "http://slack.com"},
		Enabled: true,
	}

	if err := db.CreateNotificationChannel(ctx, nc); err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}

	got, err := db.GetNotificationChannel(ctx, nc.ID)
	if err != nil || got == nil {
		t.Fatalf("failed to get notification channel: %v", err)
	}

	channels, _ := db.ListNotificationChannels(ctx)
	if len(channels) != 1 {
		t.Errorf("expected 1 channel, got %d", len(channels))
	}

	nc.Name = "Updated Slack"
	if err := db.UpdateNotificationChannel(ctx, nc); err != nil {
		t.Fatalf("failed to update notification channel: %v", err)
	}

	if err := db.DeleteNotificationChannel(ctx, nc.ID); err != nil {
		t.Fatalf("failed to delete notification channel: %v", err)
	}
}

func TestHeartbeatQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	h := &models.Heartbeat{
		Slug:      "test-slug",
		MonitorID: "mon-1",
		ExpectedS: 300,
		GraceS:    600,
	}

	if err := db.UpsertHeartbeat(ctx, h); err != nil {
		t.Fatalf("failed to upsert heartbeat: %v", err)
	}

	got, _ := db.GetHeartbeat(ctx, "test-slug")
	if got == nil || got.MonitorID != h.MonitorID {
		t.Errorf("expected heartbeat for mon-1, got %v", got)
	}
}

func TestCheckResultQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	monitorID := "mon-1"

	r := &models.CheckResult{
		MonitorID: monitorID,
		Status:    models.StatusUp,
		LatencyMs: func(i int) *int { return &i }(100),
		CheckedAt: time.Now(),
	}

	if err := db.InsertCheckResult(ctx, r); err != nil {
		t.Fatalf("failed to insert check result: %v", err)
	}

	results, err := db.GetRecentChecks(ctx, monitorID, 10)
	if err != nil {
		t.Fatalf("failed to get recent checks: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	} else if results[0].MonitorID != monitorID {
		t.Errorf("expected monitor ID %s, got %s", monitorID, results[0].MonitorID)
	}
}

func TestCheckAggregateQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	floatPtr := func(f float64) *float64 { return &f }
	a := &models.CheckAggregate{
		MonitorID:   "mon-1",
		PeriodStart: time.Now().Truncate(time.Hour),
		Resolution:  "1h",
		TotalChecks: 10,
		UpCount:     9,
		DownCount:   1,
		UptimePct:   floatPtr(90.0),
	}

	if err := db.UpsertCheckAggregate(ctx, a); err != nil {
		t.Fatalf("failed to upsert aggregate: %v", err)
	}

	// Update it
	a.UpCount = 10
	a.DownCount = 0
	a.UptimePct = floatPtr(100.0)
	if err := db.UpsertCheckAggregate(ctx, a); err != nil {
		t.Fatalf("failed to update aggregate: %v", err)
	}
}

func TestSummaryQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	monitorID := "mon-1"

	// Create monitor and some checks
	m := &models.Monitor{
		ID:        monitorID,
		Name:      "Test",
		Type:      "http",
		Config:    json.RawMessage(`{}`),
		CreatedBy: "user-1",
	}
	if err := db.CreateMonitor(ctx, m); err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	for i := 0; i < 5; i++ {
		r := &models.CheckResult{
			MonitorID: monitorID,
			Status:    models.StatusUp,
			CheckedAt: time.Now().Add(time.Duration(-i) * time.Minute),
		}
		if err := db.InsertCheckResult(ctx, r); err != nil {
			t.Fatalf("failed to insert check result: %v", err)
		}
	}

	// 1. GetLatestCheck
	latest, err := db.GetLatestCheck(ctx, monitorID)
	if err != nil || latest == nil {
		t.Fatalf("failed to get latest check: %v", err)
	}

	// 2. GetUptimePercent
	uptime, err := db.GetUptimePercent(ctx, monitorID, time.Now().Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("failed to get uptime: %v", err)
	}
	if uptime != 100.0 {
		t.Errorf("expected 100%% uptime, got %f", uptime)
	}

	// 3. GetMonitorsSummary
	summaries, err := db.GetMonitorsSummary(ctx)
	if err != nil {
		t.Fatalf("failed to get summaries: %v", err)
	}
	if len(summaries) != 1 {
		t.Errorf("expected 1 summary, got %d", len(summaries))
	}
}

func TestPurgeOldChecks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	monitorID := "mon-1"

	// Create some old and new checks
	for i := 0; i < 10; i++ {
		r := &models.CheckResult{
			MonitorID: monitorID,
			Status:    models.StatusUp,
			CheckedAt: time.Now().Add(time.Duration(-i) * 24 * time.Hour),
		}
		db.InsertCheckResult(ctx, r)
	}

	purged, err := db.PurgeOldChecks(ctx, time.Now().Add(-5*24*time.Hour))
	if err != nil {
		t.Fatalf("failed to purge checks: %v", err)
	}
	if purged < 4 {
		t.Errorf("expected at least 4 checks purged, got %d", purged)
	}
}

func TestStorage_MoreQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 1. GetMonitorTags
	m := &models.Monitor{
		ID:        "mon-tags",
		Name:      "TagsTest",
		Type:      "http",
		Tags:      []string{"tag1", "tag2"},
		Config:    json.RawMessage(`{}`),
		CreatedBy: "user-1",
	}
	if err := db.CreateMonitor(ctx, m); err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}
	tags, err := db.GetMonitorTags(ctx, "mon-tags")
	if err != nil || len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(tags), err)
	}

	// 2. GetIncident
	i := &models.Incident{ID: "inc-1", Title: "Test", Status: "active", StartedAt: time.Now(), CreatedBy: "user-1"}
	db.CreateIncident(ctx, i)
	gotInc, err := db.GetIncident(ctx, "inc-1")
	if err != nil || gotInc == nil {
		t.Errorf("failed to get incident: %v", err)
	}

	// 3. GetMaintenanceWindow
	mw := &models.MaintenanceWindow{ID: "mw-1", Title: "Test", MonitorIDs: []string{"mon-1"}, StartsAt: time.Now(), EndsAt: time.Now().Add(time.Hour), CreatedBy: "user-1"}
	db.CreateMaintenanceWindow(ctx, mw)
	gotMW, err := db.GetMaintenanceWindow(ctx, "mw-1")
	if err != nil || gotMW == nil {
		t.Errorf("failed to get maintenance window: %v", err)
	}
}

func TestStatusPageQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	sp := &models.StatusPage{
		ID:       "sp-1",
		Name:     "Status",
		Slug:     "status",
		IsPublic: true,
	}

	if err := db.CreateStatusPage(ctx, sp); err != nil {
		t.Fatalf("failed to create status page: %v", err)
	}

	got, err := db.GetStatusPageByID(ctx, "sp-1")
	if err != nil || got == nil {
		t.Fatalf("failed to get status page by ID: %v", err)
	}

	got, err = db.GetStatusPageBySlug(ctx, "status")
	if err != nil || got == nil {
		t.Fatalf("failed to get status page by slug: %v", err)
	}

	sp.Name = "Updated"
	if err := db.UpdateStatusPage(ctx, sp); err != nil {
		t.Fatalf("failed to update status page: %v", err)
	}

	pages, _ := db.ListStatusPages(ctx)
	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}

	if err := db.DeleteStatusPage(ctx, "sp-1"); err != nil {
		t.Fatalf("failed to delete status page: %v", err)
	}
}

func TestGroupQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	m1 := &models.Monitor{
		ID:        "m1",
		Name:      "M1",
		Type:      "http",
		Groups:    []string{"Group A"},
		Config:    []byte("{}"),
		CreatedBy: "u1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m2 := &models.Monitor{
		ID:        "m2",
		Name:      "M2",
		Type:      "http",
		Groups:    []string{"Group A"},
		Config:    []byte("{}"),
		CreatedBy: "u1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.CreateMonitor(ctx, m1); err != nil {
		t.Fatalf("failed to create m1: %v", err)
	}
	if err := db.CreateMonitor(ctx, m2); err != nil {
		t.Fatalf("failed to create m2: %v", err)
	}

	// Verify creation
	check, _ := db.GetMonitor(ctx, "m1")
	if check == nil {
		t.Fatal("m1 was not found immediately after creation")
	}

	// List
	groups, err := db.ListGroups(ctx)
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	if len(groups) != 1 || groups[0] != "Group A" {
		t.Errorf("expected Group A, got %v", groups)
	}

	// Rename
	if err := db.RenameGroup(ctx, "Group A", "Group B"); err != nil {
		t.Fatalf("RenameGroup failed: %v", err)
	}
	got, err := db.GetMonitor(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMonitor m1 failed after rename: %v", err)
	}
	if got == nil {
		t.Fatal("m1 missing after rename")
	}
	if len(got.Groups) == 0 || got.Groups[0] != "Group B" {
		t.Errorf("expected Group B, got %v", got.Groups)
	}

	// Delete
	if err := db.DeleteGroup(ctx, "Group B"); err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}
	got, err = db.GetMonitor(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMonitor m1 failed after delete group: %v", err)
	}
	if got == nil {
		t.Fatal("m1 missing after delete group")
	}
	if len(got.Groups) != 0 {
		t.Errorf("expected empty groups, got %v", got.Groups)
	}
}

func TestAdminUserQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	u1 := &models.User{ID: "u1", Username: "alice", Role: models.RoleViewer, CreatedAt: time.Now()}
	if err := db.CreateUser(ctx, u1); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// List
	users, _ := db.ListUsers(ctx)
	found := false
	for _, u := range users {
		if u.Username == "alice" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected alice in list")
	}

	// Update Role
	db.UpdateUserRole(ctx, "u1", models.RoleAdmin)
	got, _ := db.GetUserByID(ctx, "u1")
	if got.Role != models.RoleAdmin {
		t.Errorf("expected admin, got %s", got.Role)
	}

	// Delete
	db.DeleteUser(ctx, "u1")
	got, _ = db.GetUserByID(ctx, "u1")
	if got != nil {
		t.Error("expected user to be deleted")
	}
}

func TestSettingsQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Set/Get
	db.SetSetting(ctx, "theme", "dark")
	val, _ := db.GetSetting(ctx, "theme")
	if val != "dark" {
		t.Errorf("expected dark, got %s", val)
	}

	// Update
	db.SetSetting(ctx, "theme", "light")
	val, _ = db.GetSetting(ctx, "theme")
	if val != "light" {
		t.Errorf("expected light, got %s", val)
	}

	// List
	db.SetSetting(ctx, "site_name", "updu")
	settings, _ := db.ListSettings(ctx)
	if len(settings) != 2 {
		t.Errorf("expected 2 settings, got %d", len(settings))
	}
}

func TestSessionQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Need a user first because of foreign key
	u := &models.User{
		ID:        "user-session-test",
		Username:  "sessiontester",
		Role:      models.RoleAdmin,
		CreatedAt: time.Now(),
	}
	db.CreateUser(ctx, u)

	s1 := &models.Session{
		ID:        "sess-1",
		UserID:    u.ID,
		UserAgent: "Mozilla/5.0",
		IPAddr:    "127.0.0.1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := db.CreateSession(ctx, s1); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	got, err := db.GetSession(ctx, "sess-1")
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if got == nil || got.UserID != u.ID {
		t.Fatalf("session mistmatch or not found")
	}

	// Test CleanExpiredSessions
	s2 := &models.Session{
		ID:        "sess-expired",
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
		CreatedAt: time.Now(),
	}
	db.CreateSession(ctx, s2)

	if err := db.CleanExpiredSessions(ctx); err != nil {
		t.Fatalf("failed to clean expired sessions: %v", err)
	}
	gotExp, _ := db.GetSession(ctx, "sess-expired")
	if gotExp != nil {
		t.Fatalf("expected expired session to be cleaned or not returned")
	}

	// Delete
	if err := db.DeleteSession(ctx, "sess-1"); err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}
	gotDel, _ := db.GetSession(ctx, "sess-1")
	if gotDel != nil {
		t.Fatalf("expected session to be deleted")
	}
}

func TestGetSystemMetrics(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initially empty
	metrics, err := db.GetSystemMetrics(ctx)
	if err != nil {
		t.Fatalf("failed to get empty system metrics: %v", err)
	}
	if metrics.TotalMonitors != 0 {
		t.Errorf("expected 0 monitors, got %d", metrics.TotalMonitors)
	}

	u := &models.User{ID: "metrics-user", Username: "metrics", Role: models.RoleAdmin, CreatedAt: time.Now()}
	db.CreateUser(ctx, u)

	// Insert monitors
	m1 := &models.Monitor{ID: "m1", Name: "M1", Type: "http", Config: []byte(`{}`), Enabled: true, CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m2 := &models.Monitor{ID: "m2", Name: "M2", Type: "http", Config: []byte(`{}`), Enabled: false, CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := db.CreateMonitor(ctx, m1); err != nil {
		t.Fatalf("failed m1: %v", err)
	}
	if err := db.CreateMonitor(ctx, m2); err != nil {
		t.Fatalf("failed m2: %v", err)
	}

	lat := 10
	db.InsertCheckResult(ctx, &models.CheckResult{MonitorID: "m1", Status: models.StatusUp, LatencyMs: &lat, CheckedAt: time.Now()})

	// Add incident
	db.CreateIncident(ctx, &models.Incident{ID: "inc-1", Title: "Outage", Status: "investigating", StartedAt: time.Now(), CreatedBy: "admin"})

	metrics, err = db.GetSystemMetrics(ctx)
	if err != nil {
		t.Fatalf("failed to get populated system metrics: %v", err)
	}
	if metrics.TotalMonitors != 2 {
		t.Errorf("expected 2 total monitors, got %d", metrics.TotalMonitors)
	}
	if metrics.MonitorsPaused != 1 {
		t.Errorf("expected 1 paused monitor, got %d", metrics.MonitorsPaused)
	}
	if metrics.MonitorsUp != 1 {
		t.Errorf("expected 1 up monitor, got %d", metrics.MonitorsUp)
	}
	if metrics.ActiveIncidents != 1 {
		t.Errorf("expected 1 active incident, got %d", metrics.ActiveIncidents)
	}
}
