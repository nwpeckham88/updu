package notifier

import (
	"context"
	"log/slog"
	"sync"

	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/storage"
)

// Channel defines the interface for notification delivery.
type Channel interface {
	Type() string
	Send(ctx context.Context, monitor *models.Monitor, result *models.CheckResult, config map[string]any) error
}

// Notifier handles dispatching alerts to various channels.
type Notifier struct {
	db       *storage.DB
	channels map[string]Channel
	mu       sync.RWMutex
}

// New creates a new Notifier.
func New(db *storage.DB) *Notifier {
	return &Notifier{
		db:       db,
		channels: make(map[string]Channel),
	}
}

// Register adds a notification channel implementation.
func (n *Notifier) Register(c Channel) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.channels[c.Type()] = c
}

// Notify dispatches an alert for a monitor state change.
func (n *Notifier) Notify(ctx context.Context, monitor *models.Monitor, result *models.CheckResult) {
	// Fetch enabled notification channels
	allChannels, err := n.db.ListNotificationChannels(ctx)
	if err != nil {
		slog.Error("failed to list notification channels", "error", err)
		return
	}

	for _, nc := range allChannels {
		if !nc.Enabled {
			continue
		}

		n.mu.RLock()
		impl, ok := n.channels[nc.Type]
		n.mu.RUnlock()

		if !ok {
			slog.Warn("unknown notification channel type", "type", nc.Type)
			continue
		}

		// Run in goroutine to not block the scheduler
		go func(nc *models.NotificationChannel, impl Channel) {
			slog.Info("sending notification", "channel", nc.Name, "type", nc.Type, "monitor", monitor.Name, "status", result.Status)
			if err := impl.Send(ctx, monitor, result, nc.Config); err != nil {
				slog.Error("failed to send notification", "channel", nc.Name, "error", err)
			}
		}(nc, impl)
	}
}
