package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/diagnostic"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/notifier"
	"github.com/updu/updu/internal/notifier/channels"
	"github.com/updu/updu/internal/realtime"
	"github.com/updu/updu/internal/storage"
)

const (
	addMonitorDelayCap = 5 * time.Second
	startupSpreadCap   = 30 * time.Second
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
	randFloat64    func() float64
	now            func() time.Time
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
		randFloat64:    rand.Float64,
		now:            time.Now,
	}
}

// Start loads all monitors and begins scheduling checks.
func (s *Scheduler) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	monitors, err := s.db.ListMonitors(s.ctx)
	if err != nil {
		return err
	}

	enabledMonitors := make([]*models.Monitor, 0, len(monitors))
	for _, m := range monitors {
		if m.Enabled {
			enabledMonitors = append(enabledMonitors, m)
		}
	}

	slog.Info("starting scheduler", "monitors", len(enabledMonitors))

	for position, m := range enabledMonitors {
		s.scheduleStartupMonitor(ctx, m, position, len(enabledMonitors))
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

// MonitorCount returns how many monitors are currently scheduled.
func (s *Scheduler) MonitorCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.monitors)
}

// RunCheckSync performs a check synchronously (for testing).
func (s *Scheduler) RunCheckSync(ctx context.Context, m *models.Monitor) {
	s.runCheck(ctx, m)
}

func (s *Scheduler) scheduleMonitor(ctx context.Context, m *models.Monitor) {
	s.scheduleMonitorWithPosition(ctx, m, false, 0, 1)
}

func (s *Scheduler) scheduleStartupMonitor(ctx context.Context, m *models.Monitor, position, total int) {
	s.scheduleMonitorWithPosition(ctx, m, true, position, total)
}

func (s *Scheduler) scheduleMonitorWithPosition(ctx context.Context, m *models.Monitor, startup bool, position, total int) {
	s.mu.Lock()
	monCtx := s.ctx

	ms := &monitorState{
		monitor:    m,
		lastStatus: models.StatusPending,
	}
	s.monitors[m.ID] = ms
	s.mu.Unlock()

	tickInterval := s.monitorTickInterval(m)
	firstRun := s.initialDelay(ctx, m, tickInterval, startup, position, total)

	if monCtx == nil {
		monCtx = ctx
	}

	s.loopWg.Add(1)
	go func() {
		defer s.loopWg.Done()

		if firstRun > 0 {
			timer := time.NewTimer(firstRun)
			defer timer.Stop()
			select {
			case <-monCtx.Done():
				return
			case <-timer.C:
			}
		} else {
			select {
			case <-monCtx.Done():
				return
			default:
			}
		}

		s.wg.Add(1)
		s.runCheck(monCtx, m)
		s.wg.Done()

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

func (s *Scheduler) monitorTickInterval(m *models.Monitor) time.Duration {
	interval := time.Duration(m.IntervalS) * time.Second
	if interval <= 0 {
		interval = time.Millisecond
	}
	if s.DisableStagger {
		return interval
	}

	// #nosec G404 -- jitter/stagger doesn't need to be cryptographically secure
	jitter := time.Duration(float64(interval) * (s.randomFloat64()*0.2 - 0.1))
	tickInterval := interval + jitter
	if tickInterval <= 0 {
		return time.Millisecond
	}
	return tickInterval
}

func (s *Scheduler) initialDelay(ctx context.Context, m *models.Monitor, tickInterval time.Duration, startup bool, position, total int) time.Duration {
	if s.DisableStagger {
		return 0
	}

	if startup {
		latest, err := s.db.GetLatestCheck(ctx, m.ID)
		if err != nil {
			slog.Warn("loading latest check for startup scheduling", "monitor", m.Name, "error", err)
		} else if latest != nil {
			nextRun := latest.CheckedAt.Add(tickInterval)
			remaining := nextRun.Sub(s.currentTime())
			if remaining > 0 {
				return remaining
			}
		}
		return s.startupDelay(tickInterval, position, total)
	}

	maxDelay := tickInterval
	if maxDelay > addMonitorDelayCap {
		maxDelay = addMonitorDelayCap
	}
	return s.randomDuration(maxDelay)
}

func (s *Scheduler) startupDelay(tickInterval time.Duration, position, total int) time.Duration {
	if total <= 1 {
		return 0
	}

	window := tickInterval
	if window > startupSpreadCap {
		window = startupSpreadCap
	}
	if window <= 0 {
		return 0
	}

	slotWidth := window / time.Duration(total)
	if slotWidth <= 0 {
		return s.randomDuration(window)
	}

	base := time.Duration(position) * slotWidth
	slotSpan := slotWidth
	if position >= total-1 {
		slotSpan = window - base
	}
	return base + s.randomDuration(slotSpan)
}

func (s *Scheduler) randomDuration(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(s.randomFloat64() * float64(max))
}

func (s *Scheduler) randomFloat64() float64 {
	if s.randFloat64 != nil {
		return s.randFloat64()
	}
	return rand.Float64()
}

func (s *Scheduler) currentTime() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
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

	// Update corresponding service endpoint check and harvest TLS certificates
	defaultEndpointID := m.ID + "-default"
	ec := &models.EndpointCheck{
		ServiceID:  m.ID,
		EndpointID: defaultEndpointID,
		NodeID:     "local",
		Status:     result.Status,
		LatencyMs:  result.LatencyMs,
		StatusCode: result.StatusCode,
		Message:    result.Message,
		Metadata:   result.Metadata,
		CheckedAt:  result.CheckedAt,
	}
	_ = s.db.RecordEndpointCheck(ctx, ec)

	if len(result.Metadata) > 0 {
		if cert := checker.ExtractTLSCertificate(result.Metadata, &defaultEndpointID, m.Name); cert != nil {
			_ = s.db.UpsertTLSCertificate(ctx, cert)
		}
	}

	// Multi-endpoint service probing & differential root-cause diagnostics
	diag := s.probeServiceEndpoints(ctx, m, ec)

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
		msg := result.Message
		if diag != nil && diag.ProbableCause != "" && result.Status != models.StatusUp {
			if msg != "" {
				msg = fmt.Sprintf("[%s] %s (%s)", diag.Summary, diag.ProbableCause, msg)
			} else {
				msg = fmt.Sprintf("[%s] %s", diag.Summary, diag.ProbableCause)
			}
		}

		// Create and store an Event for the transition
		event := &models.Event{
			MonitorID: m.ID,
			Status:    result.Status,
			Message:   msg,
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
			notifyCtx := ctx
			if s.allowLocalhost {
				notifyCtx = context.WithValue(notifyCtx, channels.AllowLocalhostKey, true)
			}
			s.notifier.Notify(notifyCtx, m, event)
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

func (s *Scheduler) probeServiceEndpoints(ctx context.Context, m *models.Monitor, primaryCheck *models.EndpointCheck) *diagnostic.ServiceDiagnosis {
	endpoints, err := s.db.ListServiceEndpoints(ctx, m.ID)
	if err != nil || len(endpoints) == 0 {
		return nil
	}

	checksMap := make(map[string]*models.EndpointCheck)
	defaultEndpointID := m.ID + "-default"
	checksMap[defaultEndpointID] = primaryCheck

	for _, ep := range endpoints {
		if ep.ID == defaultEndpointID {
			continue
		}

		c := s.registry.Get(ep.TargetType)
		if c == nil {
			continue
		}

		timeout := ep.TimeoutS
		if timeout <= 0 {
			timeout = 10
		}
		interval := ep.IntervalS
		if interval <= 0 {
			interval = 30
		}

		tempMon := &models.Monitor{
			ID:        ep.ID,
			Name:      ep.Name,
			Type:      ep.TargetType,
			Config:    ep.Config,
			TimeoutS:  timeout,
			IntervalS: interval,
		}

		epCtx := ctx
		if s.allowLocalhost {
			epCtx = context.WithValue(epCtx, checker.AllowLocalhostKey, true)
		}
		epCtx, epCancel := context.WithTimeout(epCtx, time.Duration(timeout)*time.Second)

		start := time.Now()
		checkRes, err := c.Check(epCtx, tempMon)
		duration := int(time.Since(start).Milliseconds())
		epCancel()

		epCheck := &models.EndpointCheck{
			ServiceID:  m.ID,
			EndpointID: ep.ID,
			NodeID:     "local",
			CheckedAt:  time.Now(),
			LatencyMs:  &duration,
		}

		if err != nil || checkRes == nil {
			epCheck.Status = models.StatusDown
			if err != nil {
				epCheck.Message = err.Error()
			}
		} else {
			epCheck.Status = checkRes.Status
			epCheck.LatencyMs = checkRes.LatencyMs
			epCheck.StatusCode = checkRes.StatusCode
			epCheck.Message = checkRes.Message
			epCheck.Metadata = checkRes.Metadata

			if len(checkRes.Metadata) > 0 {
				if cert := checker.ExtractTLSCertificate(checkRes.Metadata, &ep.ID, ep.Name); cert != nil {
					_ = s.db.UpsertTLSCertificate(ctx, cert)
				}
			}
		}

		_ = s.db.RecordEndpointCheck(ctx, epCheck)
		checksMap[ep.ID] = epCheck
	}

	svc, err := s.db.GetService(ctx, m.ID)
	if err != nil || svc == nil {
		return nil
	}

	diag := diagnostic.EvaluateServiceHealth(svc, checksMap)
	s.sse.Broadcast(realtime.Event{
		Type: "service:status",
		Data: map[string]any{
			"id":              svc.ID,
			"name":            svc.Name,
			"status":          diag.Status,
			"diagnosis":       diag.Summary,
			"probable_cause":  diag.ProbableCause,
			"action_hint":     diag.ActionHint,
			"healthy_count":   diag.HealthyCount,
			"total_count":     diag.TotalCount,
			"probe_breakdown": diag.ProbeBreakdown,
		},
	})

	return diag
}

