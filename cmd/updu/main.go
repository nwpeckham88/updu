package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/updu/updu/internal/api"
	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/config"
	"github.com/updu/updu/internal/notifier"
	"github.com/updu/updu/internal/notifier/channels"
	"github.com/updu/updu/internal/realtime"
	"github.com/updu/updu/internal/scheduler"
	"github.com/updu/updu/internal/storage"
	"github.com/updu/updu/internal/updater"
)

//go:embed all:frontend/build
var frontendFS embed.FS

func main() {
	// Handle subcommands (install, uninstall, version, help)
	if handleSubcommand() {
		return
	}

	// 2. Load configuration
	cfg := config.Load()

	// 1. Setup structured logging (after config so we can read log level)
	var logLevel slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)
	slog.Info("starting updu...", "log_level", cfg.LogLevel)

	// 3. Initialize SQLite (WAL mode, Pi Zero W tuned)
	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate(context.Background()); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// 3.5 GitOps: Sync monitors if updu.conf is present
	if cfg.ConfigPath != "" {
		slog.Info("gitops: syncing monitors from config", "path", cfg.ConfigPath)
		yCfg, err := config.ParseYAMLConfig(cfg.ConfigPath)
		if err != nil {
			slog.Error("gitops: failed to parse config", "error", err)
		} else {
			monitors, err := yCfg.ToModels()
			if err != nil {
				slog.Error("gitops: failed to convert monitors", "error", err)
			} else {
				if err := db.SyncMonitors(context.Background(), monitors); err != nil {
					slog.Error("gitops: failed to sync monitors", "error", err)
				} else {
					slog.Info("gitops: sync complete", "count", len(monitors))
				}
			}
		}
	}

	// 4. Background tasks (Cleanup)
	cleanupDone := make(chan struct{})
	cleanupStop := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		sessionTicker := time.NewTicker(1 * time.Hour)
		purgeTicker := time.NewTicker(24 * time.Hour)
		defer sessionTicker.Stop()
		defer purgeTicker.Stop()

		for {
			select {
			case <-cleanupStop:
				return
			case <-sessionTicker.C:
				db.CleanExpiredSessions(context.Background())
			case <-purgeTicker.C:
				// Purge checks older than 30 days
				olderThan := time.Now().AddDate(0, 0, -30)
				count, err := db.PurgeOldChecks(context.Background(), olderThan)
				if err == nil && count > 0 {
					slog.Info("purged old checks", "count", count, "older_than", olderThan)
				}
			}
		}
	}()
	defer func() { close(cleanupStop); <-cleanupDone }()

	// 5. Initialize Aggregator
	agg := storage.NewAggregator(db, 5*time.Minute)
	agg.Start(context.Background())
	defer agg.Stop()

	// 6. Initialize Auth
	a := auth.New(db, cfg)
	if err := a.EnsureFirstUser(context.Background()); err != nil {
		slog.Error("failed to ensure first user", "error", err)
	}

	// 7. Initialize Checkers Registry
	reg := checker.NewRegistry(cfg.AllowLocalhost, db)

	// 8. Initialize SSE Hub
	sse := realtime.NewHub()

	// 9. Initialize Notifier
	n := notifier.New(db)
	n.Register(channels.NewWebhookChannel())
	n.Register(channels.NewDiscordChannel())
	n.Register(channels.NewSlackChannel())
	n.Register(channels.NewEmailChannel())
	n.Register(channels.NewNtfyChannel())

	// 10. Initialize and Start Scheduler
	sched := scheduler.New(db, reg, sse, n, cfg.WorkerPoolSize)
	if err := sched.Start(context.Background()); err != nil {
		slog.Error("failed to start scheduler", "error", err)
		os.Exit(1)
	}
	defer sched.Stop()

	// 11. Initialize API Router
	server := api.NewServer(db, a, reg, sched, n, sse, cfg)
	apiRouter := server.Router()

	// 12. Mount API and Static Frontend
	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)
	mux.Handle("/heartbeat/", apiRouter)
	mux.Handle("/healthz", apiRouter) // Top-level health check for load balancers / Docker / k8s

	// Serve the static SPA from embedded FS, falling back to index.html for routing
	staticFS, err := fs.Sub(frontendFS, "frontend/build")
	if err != nil {
		slog.Error("failed to create static fs", "error", err)
	} else {
		fileServer := http.FileServer(http.FS(staticFS))
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if path == "/" {
				path = "index.html"
			} else {
				path = strings.TrimPrefix(path, "/")
			}

			if f, err := staticFS.Open(path); err != nil {
				// Fallback to index.html for client-side routing
				r.URL.Path = "/"
			} else {
				f.Close()
			}
			fileServer.ServeHTTP(w, r)
		}))
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 11. Start HTTP Server
	go func() {
		slog.Info("listening on", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	// 12. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}

	// If a self-update scheduled this shutdown, exit non-zero so process
	// supervisors (systemd Restart=on-failure, docker restart=unless-stopped,
	// runit, s6) relaunch us on the freshly-installed binary.
	if updater.RestartRequested() {
		slog.Info("self-update restart: exiting for supervisor relaunch",
			"reason", updater.RestartReason(),
			"exit_code", updater.RestartExitCode,
		)
		os.Exit(updater.RestartExitCode)
	}
}
