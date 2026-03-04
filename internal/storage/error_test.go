package storage

import (
	"context"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestStorage_ErrorsWhenClosed(t *testing.T) {
	db, cleanup := setupTestDB(t)

	// Close the DB immediately to trigger errors on all queries
	cleanup()

	ctx := context.Background()

	// Users
	_, _ = db.GetUserByUsername(ctx, "test")
	_, _ = db.ListUsers(ctx)
	_ = db.CreateUser(ctx, &models.User{})
	_ = db.UpdateUserRole(ctx, "id", "admin")
	_ = db.DeleteUser(ctx, "id")

	// Sessions
	_ = db.CreateSession(ctx, &models.Session{})
	_, _ = db.GetSession(ctx, "id")
	_ = db.DeleteSession(ctx, "id")
	_ = db.CleanExpiredSessions(ctx)

	// Monitors
	_ = db.CreateMonitor(ctx, &models.Monitor{})
	_, _ = db.GetMonitor(ctx, "id")
	_, _ = db.ListMonitors(ctx)
	_ = db.UpdateMonitor(ctx, &models.Monitor{})
	_ = db.DeleteMonitor(ctx, "id")
	_, _ = db.GetMonitorsSummary(ctx)

	// Groups
	_, _ = db.ListGroups(ctx)
	_ = db.RenameGroup(ctx, "old", "new")
	_ = db.DeleteGroup(ctx, "name")

	// Checks
	_ = db.InsertCheckResult(ctx, &models.CheckResult{})
	_, _ = db.GetRecentChecks(ctx, "id", 10)
	_, _ = db.GetLatestCheck(ctx, "id")
	_, _ = db.GetUptimePercent(ctx, "id", time.Now())
	_, _ = db.GetCheckAggregateStats(ctx, "id", time.Now(), time.Now())
	_, _ = db.PurgeOldChecks(ctx, time.Now())

	// Status Pages
	_ = db.CreateStatusPage(ctx, &models.StatusPage{})
	_, _ = db.ListStatusPages(ctx)
	_, _ = db.GetStatusPageBySlug(ctx, "slug")
	_, _ = db.GetStatusPageByID(ctx, "id")
	_ = db.UpdateStatusPage(ctx, &models.StatusPage{})
	_ = db.DeleteStatusPage(ctx, "id")

	// Notifications
	_ = db.CreateNotificationChannel(ctx, &models.NotificationChannel{})
	_, _ = db.ListNotificationChannels(ctx)
	_, _ = db.GetNotificationChannel(ctx, "id")
	_ = db.UpdateNotificationChannel(ctx, &models.NotificationChannel{})
	_ = db.DeleteNotificationChannel(ctx, "id")

	// Incidents
	_ = db.CreateIncident(ctx, &models.Incident{})
	_, _ = db.ListIncidents(ctx)
	_, _ = db.GetIncident(ctx, "id")
	_ = db.UpdateIncident(ctx, &models.Incident{})
	_ = db.DeleteIncident(ctx, "id")

	// Maintenance
	_ = db.CreateMaintenanceWindow(ctx, &models.MaintenanceWindow{})
	_, _ = db.ListMaintenanceWindows(ctx)
	_, _ = db.GetMaintenanceWindow(ctx, "id")
	_ = db.UpdateMaintenanceWindow(ctx, &models.MaintenanceWindow{})
	_ = db.DeleteMaintenanceWindow(ctx, "id")
	_, _ = db.IsMonitorUnderMaintenance(ctx, "id")

	// Heartbeats
	_ = db.UpsertHeartbeat(ctx, &models.Heartbeat{})
	_, _ = db.GetHeartbeat(ctx, "id")

	// Settings
	_, _ = db.ListSettings(ctx)
	_, _ = db.GetSetting(ctx, "key")
	_ = db.SetSetting(ctx, "key", "val")

	// System Metrics
	_, _ = db.GetSystemMetrics(ctx)

	// Aggregates
	_ = db.UpsertCheckAggregate(ctx, &models.CheckAggregate{})
}
