package checker

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/updu/updu/internal/models"
)

type stubStatusReader struct {
	statuses map[string]models.MonitorStatus
	err      error
}

func (s *stubStatusReader) GetMonitorStatuses(ctx context.Context, ids []string) (map[string]models.MonitorStatus, error) {
	return s.statuses, s.err
}

func TestCompositeChecker(t *testing.T) {
	c := &CompositeChecker{}
	if c.Type() != "composite" {
		t.Fatalf("Type() = %q, want %q", c.Type(), "composite")
	}

	if err := c.Validate([]byte(`{"monitor_ids":["a","b"],"mode":"all_up"}`)); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
	if err := c.Validate([]byte(`{"monitor_ids":[],"mode":"all_up"}`)); err == nil {
		t.Fatal("expected error for empty monitor_ids")
	}
	if err := c.Validate([]byte(`{"monitor_ids":["a"],"mode":"quorum","quorum":0}`)); err == nil {
		t.Fatal("expected error for quorum <= 0")
	}
	if err := c.Validate([]byte(`{"monitor_ids":["a"],"mode":"bad"}`)); err == nil {
		t.Fatal("expected error for bad mode")
	}

	ctx := context.Background()

	t.Run("reports up when quorum is met", func(t *testing.T) {
		checker := &CompositeChecker{sr: &stubStatusReader{statuses: map[string]models.MonitorStatus{
			"a": models.StatusUp,
			"b": models.StatusUp,
			"c": models.StatusDown,
		}}}

		monitor := &models.Monitor{
			ID: "comp-quorum-up",
			Config: json.RawMessage(`{
				"monitor_ids":["a","b","c"],
				"mode":"quorum",
				"quorum":2
			}`),
		}

		res, err := checker.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusUp {
			t.Fatalf("expected up, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "2/3 monitors up") {
			t.Fatalf("expected quorum summary, got %q", res.Message)
		}
		if len(res.Metadata) == 0 {
			t.Fatal("expected metadata to be populated")
		}
	})

	t.Run("reports up when any_up finds one healthy dependency", func(t *testing.T) {
		checker := &CompositeChecker{sr: &stubStatusReader{statuses: map[string]models.MonitorStatus{
			"a": models.StatusPending,
			"b": models.StatusUp,
		}}}

		monitor := &models.Monitor{
			ID:     "comp-any-up",
			Config: json.RawMessage(`{"monitor_ids":["a","b"],"mode":"any_up"}`),
		}

		res, err := checker.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusUp {
			t.Fatalf("expected up, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "1/2 monitors up") {
			t.Fatalf("expected any_up success message, got %q", res.Message)
		}
	})

	t.Run("reports down when all_up requirement fails", func(t *testing.T) {
		checker := &CompositeChecker{sr: &stubStatusReader{statuses: map[string]models.MonitorStatus{
			"a": models.StatusUp,
			"b": models.StatusDown,
		}}}

		monitor := &models.Monitor{
			ID:     "comp-all-up-down",
			Config: json.RawMessage(`{"monitor_ids":["a","b"],"mode":"all_up"}`),
		}

		res, err := checker.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusDown {
			t.Fatalf("expected down, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "all required") {
			t.Fatalf("expected all_up failure message, got %q", res.Message)
		}
	})

	t.Run("reports down when the status reader errors", func(t *testing.T) {
		checker := &CompositeChecker{sr: &stubStatusReader{err: context.DeadlineExceeded}}
		monitor := &models.Monitor{
			ID:     "comp-reader-error",
			Config: json.RawMessage(`{"monitor_ids":["a"],"mode":"all_up"}`),
		}

		res, err := checker.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusDown {
			t.Fatalf("expected down, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "reading monitor statuses") {
			t.Fatalf("expected reader error message, got %q", res.Message)
		}
	})

	t.Run("reports down when no status reader is configured", func(t *testing.T) {
		checker := &CompositeChecker{}
		monitor := &models.Monitor{
			ID:     "comp-no-reader",
			Config: json.RawMessage(`{"monitor_ids":["a"],"mode":"any_up"}`),
		}

		res, err := checker.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusDown {
			t.Fatalf("expected down, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "not initialised") {
			t.Fatalf("expected no-reader message, got %q", res.Message)
		}
	})
}
