package scheduler

import (
	"context"
	"log/slog"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/notifier"
	"github.com/updu/updu/internal/realtime"
	"github.com/updu/updu/internal/storage"
)

// Scheduler orchestrates all monitoring checks.
type Scheduler struct {
	db             *storage.DB
	registry       *checker.Registry
	sse            *realtime.Hub
	notifier       *notifier.Notifier
	poolSize       int
	allowLocalhost bool
	sem            chan struct{}

	monitors map[string]*monitorState
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	loopWg   sync.WaitGroup
	wg       sync.WaitGroup

	DisableStagger bool // For testing
}

type monitorState struct {
	monitor    *models.Monitor
	lastStatus models.MonitorStatus
	timer      *time.Timer
}

// New creates a new scheduler.
func New(db *storage.DB, registry *checker.Registry, sse *realtime.Hub, n *notifier.Notifier, poolSize int) *Scheduler {
	if poolSize <= 0 {
		poolSize = runtime.NumCPU() * 4
		if poolSize < 4 {
			poolSize = 4
		}
		if poolSize > 50 {
			poolSize = 50
		}
	}
	slog.Info("scheduler created", "worker_pool_size", poolSize)

	return &Scheduler{
		db:             db,
		registry:       registry,
		sse:            sse,
		notifier:       n,
		poolSize:       poolSize,
		allowLocalhost: registry.AllowLocalhost, // Assuming we add it to Registry
		sem:            make(chan struct{}, poolSize),
		monitors:       make(map[string]*monitorState),
	}
}

// Start loads all monitors and begins scheduling checks.
func (s *Scheduler) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	monitors, err := s.db.ListMonitors(s.ctx)
	if err != nil {
		return err
	}

	slog.Info("starting scheduler", "monitors", len(monitors))

	for _, m := range monitors {
		if !m.Enabled {
			continue
		}
		s.scheduleMonitor(ctx, m)
	}

	return nil
}

// Stop gracefully shuts down the scheduler.
func (s *Scheduler) Stop() {
	slog.Info("stopping scheduler")
	if s.cancel != nil {
		s.cancel()
	}

	s.mu.Lock()
	for _, ms := range s.monitors {
		if ms.timer != nil {
			ms.timer.Stop()
		}
	}
	s.mu.Unlock()

	s.loopWg.Wait()
	s.wg.Wait()
	slog.Info("scheduler stopped")
}

// AddMonitor adds a new monitor to the scheduler.
func (s *Scheduler) AddMonitor(ctx context.Context, m *models.Monitor) {
	s.scheduleMonitor(ctx, m)
}

// RemoveMonitor removes a monitor from the scheduler.
func (s *Scheduler) RemoveMonitor(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ms, ok := s.monitors[id]; ok {
		if ms.timer != nil {
			ms.timer.Stop()
		}
		delete(s.monitors, id)
	}
}

// ReloadMonitor updates a monitor's schedule.
func (s *Scheduler) ReloadMonitor(ctx context.Context, m *models.Monitor) {
	s.RemoveMonitor(m.ID)
	if m.Enabled {
		s.scheduleMonitor(ctx, m)
	}
}

// RunCheckSync performs a check synchronously (for testing).
func (s *Scheduler) RunCheckSync(ctx context.Context, m *models.Monitor) {
	s.runCheck(ctx, m)
}

func (s *Scheduler) scheduleMonitor(ctx context.Context, m *models.Monitor) {
	s.mu.Lock()
	monCtx := s.ctx
	defer s.mu.Unlock()

	ms := &monitorState{
		monitor:    m,
		lastStatus: models.StatusPending,
	}
	s.monitors[m.ID] = ms

	// Add jitter ±10% to prevent thundering herd
	interval := time.Duration(m.IntervalS) * time.Second
	// #nosec G404 -- jitter/stagger doesn't need to be cryptographically secure
	jitter := time.Duration(float64(interval) * (rand.Float64()*0.2 - 0.1))
	firstRun := time.Duration(rand.Float64() * float64(5*time.Second)) // stagger initial checks

	if s.DisableStagger {
		jitter = 0
		firstRun = 0
	}

	if monCtx == nil {
		monCtx = ctx
	}

	s.loopWg.Add(1)
	go func() {
		defer s.loopWg.Done()

		select {
		case <-monCtx.Done():
			return
		case <-time.After(firstRun):
		}

		s.wg.Add(1)
		s.runCheck(monCtx, m)
		s.wg.Done()

		tickInterval := interval + jitter
		if tickInterval <= 0 {
			tickInterval = 1 * time.Millisecond
		}

		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-monCtx.Done():
				return
			case <-ticker.C:
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					s.runCheck(monCtx, m)
				}()
			}
		}
	}()
}

func (s *Scheduler) runCheck(ctx context.Context, m *models.Monitor) {
	slog.Debug("runCheck started", "monitor", m.Name, "id", m.ID)
	c := s.registry.Get(m.Type)
	if c == nil {
		slog.Warn("no checker for type", "type", m.Type, "monitor", m.Name)
		return
	}

	// Acquire semaphore slot
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return
	}

	defer func() { <-s.sem }()

	checkCtx, cancel := context.WithTimeout(ctx, time.Duration(m.TimeoutS)*time.Second)
	defer cancel()

	if s.allowLocalhost {
		checkCtx = context.WithValue(checkCtx, checker.AllowLocalhostKey, true)
	}

	var result *models.CheckResult
	retries := m.Retries
	if retries <= 0 {
		retries = 1
	}

	for attempt := 0; attempt < retries; attempt++ {
		var err error
		result, err = c.Check(checkCtx, m)
		if err != nil {
			slog.Error("check error", "monitor", m.Name, "attempt", attempt+1, "error", err)
			result = &models.CheckResult{
				MonitorID: m.ID,
				Status:    models.StatusDown,
				Message:   err.Error(),
				CheckedAt: time.Now(),
			}
		}

		// If check passed, no need to retry
		if result.Status == models.StatusUp {
			break
		}

		// If more retries remain, wait briefly before retrying
		if attempt < retries-1 {
			slog.Debug("retrying check", "monitor", m.Name, "attempt", attempt+1, "max", retries)
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}

	// Store result
	if err := s.db.InsertCheckResult(ctx, result); err != nil {
		slog.Error("storing check result", "monitor", m.Name, "error", err)
	}

	// Broadcast via SSE
	s.sse.Broadcast(realtime.Event{
		Type: "monitor:status",
		Data: map[string]any{
			"id":         m.ID,
			"name":       m.Name,
			"status":     result.Status,
			"latency_ms": result.LatencyMs,
			"message":    result.Message,
			"checked_at": result.CheckedAt,
		},
	})

	// Track state change for notifications
	s.mu.Lock()
	ms, ok := s.monitors[m.ID]
	var oldStatus models.MonitorStatus
	if ok {
		oldStatus = ms.lastStatus
		ms.lastStatus = result.Status
	}
	s.mu.Unlock()

	if ok && oldStatus != result.Status && oldStatus != models.StatusPending {
		// Create and store an Event for the transition
		event := &models.Event{
			MonitorID: m.ID,
			Status:    result.Status,
			Message:   result.Message,
			CreatedAt: time.Now(),
		}
		if err := s.db.CreateEvent(ctx, event); err != nil {
			slog.Error("storing event", "monitor", m.Name, "error", err)
		}

		// Only notify if not under maintenance
		underMaintenance, err := s.db.IsMonitorUnderMaintenance(ctx, m.ID)
		if err != nil {
			slog.Error("checking maintenance status", "monitor", m.Name, "error", err)
		}
		if !underMaintenance {
			s.notifier.Notify(ctx, m, event)
		} else {
			slog.Info("skipping notification: monitor under maintenance", "monitor", m.Name)
		}
	}

	level := slog.LevelDebug
	if result.Status != models.StatusUp {
		level = slog.LevelWarn
	}
	slog.Log(ctx, level, "check complete",
		"monitor", m.Name,
		"status", result.Status,
		"latency_ms", result.LatencyMs,
	)
}
