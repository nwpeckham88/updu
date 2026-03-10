package notifier

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/storage"
)

// Channel defines the interface for notification delivery.
type Channel interface {
	Type() string
	Send(ctx context.Context, monitor *models.Monitor, event *models.Event, config map[string]any) error
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

// GetChannel returns the Channel implementation for the given type, or nil.
func (n *Notifier) GetChannel(typ string) Channel {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.channels[typ]
}

// Notify dispatches an alert for a monitor state change.
func (n *Notifier) Notify(ctx context.Context, monitor *models.Monitor, event *models.Event) {
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

		// Run in goroutine with detached context to not block the scheduler
		go func(nc *models.NotificationChannel, impl Channel) {
			// Decouple from the parent cancellation (so the notification isn't
			// aborted when the HTTP request finishes) but preserve context values
			// such as trace IDs and test hooks via WithoutCancel.
			notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			defer cancel()
			slog.Info("sending notification", "channel", nc.Name, "type", nc.Type, "monitor", monitor.Name, "status", event.Status)
			if err := impl.Send(notifyCtx, monitor, event, nc.Config); err != nil {
				slog.Error("failed to send notification", "channel", nc.Name, "error", err)
			}
		}(nc, impl)
	}
}
