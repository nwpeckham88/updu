package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRedactMonitorClearsInvestigation(t *testing.T) {
	monitor := &Monitor{
		ID:     "monitor-1",
		Name:   "Primary API",
		Type:   "http",
		Config: json.RawMessage(`{"url":"https://example.com","headers":{"Authorization":"secret"}}`),
		Investigation: &MonitorInvestigation{
			MonitorID: "monitor-1",
			Active:    true,
			UpdatedBy: "admin-user",
			UpdatedAt: time.Now(),
		},
	}

	redacted := RedactMonitor(monitor)
	if redacted.Investigation != nil {
		t.Fatalf("expected redacted monitor to clear investigation metadata")
	}
	if monitor.Investigation == nil {
		t.Fatalf("expected source monitor investigation to remain unchanged")
	}
}
