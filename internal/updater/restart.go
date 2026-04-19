package updater

import (
	"log/slog"
	"os"
	"sync"
	"syscall"
	"time"
)

// RestartExitCode is the process exit code used after an in-process self-update.
// It is non-zero so that systemd units configured with `Restart=on-failure`
// will relaunch the binary. Process supervisors that restart on any exit
// (Docker `restart: unless-stopped`, runit, s6, etc.) treat any non-zero
// status the same as zero. Value 75 == EX_TEMPFAIL from sysexits.h.
const RestartExitCode = 75

var (
	restartMu        sync.Mutex
	restartScheduled bool
	restartReason    string
)

// ScheduleRestart asks the running process to terminate so that its supervisor
// can launch the freshly downloaded binary. The actual SIGTERM is sent after
// `delay` to give the HTTP response that triggered the update time to flush
// to the client.
//
// After this call, RestartRequested() returns true; main.go reads that flag
// and exits with RestartExitCode once graceful shutdown completes. If the
// graceful shutdown stalls, the goroutine here force-exits as a backstop.
func ScheduleRestart(delay time.Duration, reason string) {
	restartMu.Lock()
	if restartScheduled {
		restartMu.Unlock()
		return
	}
	restartScheduled = true
	restartReason = reason
	restartMu.Unlock()

	slog.Info("scheduling self-restart", "delay", delay.String(), "reason", reason)

	go func() {
		time.Sleep(delay)

		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			slog.Error("self-restart: find own process failed; forcing exit", "error", err)
			os.Exit(RestartExitCode)
		}
		if err := p.Signal(syscall.SIGTERM); err != nil {
			slog.Error("self-restart: SIGTERM failed; forcing exit", "error", err)
			os.Exit(RestartExitCode)
		}

		// Backstop: if the graceful shutdown does not exit within 30s,
		// force-exit so the supervisor can relaunch us.
		time.Sleep(30 * time.Second)
		slog.Warn("self-restart: graceful shutdown timed out; forcing exit")
		os.Exit(RestartExitCode)
	}()
}

// RestartRequested reports whether ScheduleRestart has been called.
func RestartRequested() bool {
	restartMu.Lock()
	defer restartMu.Unlock()
	return restartScheduled
}

// RestartReason returns the human-readable reason recorded by ScheduleRestart.
func RestartReason() string {
	restartMu.Lock()
	defer restartMu.Unlock()
	return restartReason
}
