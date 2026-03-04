package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const serviceName = "updu"
const unitPath = "/etc/systemd/system/updu.service"

func serviceInstall() {
	if runtime.GOOS != "linux" {
		fmt.Fprintln(os.Stderr, "error: systemd service install is only supported on Linux")
		os.Exit(1)
	}

	if os.Getuid() != 0 {
		fmt.Fprintln(os.Stderr, "error: must run as root (try sudo)")
		os.Exit(1)
	}

	// Resolve the absolute path of the current binary
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not resolve executable path: %v\n", err)
		os.Exit(1)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not resolve symlinks: %v\n", err)
		os.Exit(1)
	}

	// Detect working directory (where the binary lives)
	workDir := filepath.Dir(exe)

	unit := fmt.Sprintf(`[Unit]
Description=updu - Lightweight Uptime Monitor
Documentation=https://github.com/updu/updu
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
WorkingDirectory=%s
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=updu

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=%s
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`, exe, workDir, workDir)

	// Write the unit file
	if err := os.WriteFile(unitPath, []byte(unit), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: could not write unit file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Wrote %s\n", unitPath)

	// Reload systemd
	if err := runSystemctl("daemon-reload"); err != nil {
		fmt.Fprintf(os.Stderr, "error: systemctl daemon-reload failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Reloaded systemd")

	// Enable the service
	if err := runSystemctl("enable", serviceName); err != nil {
		fmt.Fprintf(os.Stderr, "error: systemctl enable failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Enabled updu service")

	// Start the service
	if err := runSystemctl("start", serviceName); err != nil {
		fmt.Fprintf(os.Stderr, "error: systemctl start failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Started updu service")

	fmt.Println()
	fmt.Println("updu is now running as a systemd service.")
	fmt.Println("  Status:  sudo systemctl status updu")
	fmt.Println("  Logs:    sudo journalctl -u updu -f")
	fmt.Println("  Stop:    sudo systemctl stop updu")
	fmt.Println("  Remove:  sudo updu uninstall")
}

func serviceUninstall() {
	if runtime.GOOS != "linux" {
		fmt.Fprintln(os.Stderr, "error: systemd service uninstall is only supported on Linux")
		os.Exit(1)
	}

	if os.Getuid() != 0 {
		fmt.Fprintln(os.Stderr, "error: must run as root (try sudo)")
		os.Exit(1)
	}

	// Stop the service (ignore errors if not running)
	_ = runSystemctl("stop", serviceName)
	fmt.Println("✓ Stopped updu service")

	// Disable the service
	_ = runSystemctl("disable", serviceName)
	fmt.Println("✓ Disabled updu service")

	// Remove the unit file
	if _, err := os.Stat(unitPath); err == nil {
		if err := os.Remove(unitPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: could not remove %s: %v\n", unitPath, err)
			os.Exit(1)
		}
		fmt.Printf("✓ Removed %s\n", unitPath)
	} else {
		fmt.Println("  (unit file already removed)")
	}

	// Reload systemd
	if err := runSystemctl("daemon-reload"); err != nil {
		fmt.Fprintf(os.Stderr, "error: systemctl daemon-reload failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Reloaded systemd")

	fmt.Println()
	fmt.Println("updu service has been uninstalled.")
	fmt.Println("Your data is still in the working directory. Remove it manually if desired.")
}

func printUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf(`updu - Lightweight Uptime Monitor

Usage:
  %s              Start the updu server
  %s install      Install updu as a systemd service
  %s uninstall    Uninstall the updu systemd service
  %s version      Show version info

Environment Variables:
  UPDU_HOST           Listen address (default: 0.0.0.0)
  UPDU_PORT           Listen port (default: 3000)
  UPDU_DB_PATH        Database file path (default: ./updu.db)
  UPDU_LOG_LEVEL      Log level: debug, info, warn, error (default: info)
  UPDU_ADMIN_USER     Auto-create admin username on first run
  UPDU_ADMIN_PASSWORD  Auto-create admin password on first run
`, exe, exe, exe, exe)
}

func runSystemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// handleSubcommand checks os.Args for subcommands and handles them.
// Returns true if a subcommand was handled (caller should exit).
func handleSubcommand() bool {
	if len(os.Args) < 2 {
		return false
	}

	switch strings.ToLower(os.Args[1]) {
	case "install":
		serviceInstall()
		return true
	case "uninstall", "remove":
		serviceUninstall()
		return true
	case "version":
		fmt.Println("updu v0.1.0")
		return true
	case "help", "-h", "--help":
		printUsage()
		return true
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
		return true
	}
}
