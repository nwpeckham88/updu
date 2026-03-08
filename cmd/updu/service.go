package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/updu/updu/internal/config"
	"github.com/updu/updu/internal/storage"
	"github.com/updu/updu/internal/updater"
	"github.com/updu/updu/internal/version"
	"gopkg.in/yaml.v3"
)

const serviceName = "updu"
const unitPath = "/etc/systemd/system/updu.service"

var (
	osExit      = os.Exit
	runtimeGOOS = runtime.GOOS
	osGetuid    = os.Getuid
	systemctl   = runSystemctl
)

func handleConfigFetch(argURL string) {
	// Precedence: URL argument > conf_url in updu.conf > UPDU_CONF_URL env var
	url := argURL
	if url == "" {
		// Load config to check for conf_url
		cfg := config.Load()
		url = cfg.ConfURL
	}
	if url == "" {
		url = os.Getenv("UPDU_CONF_URL")
	}

	if url == "" {
		fmt.Fprintln(os.Stderr, "error: no URL provided and UPDU_CONF_URL is not set")
		osExit(1)
		return
	}

	path := os.Getenv("UPDU_CONF_PATH")
	if path == "" {
		// Try to get path from config if not in env
		cfg := config.Load()
		path = cfg.ConfPath
	}
	if path == "" {
		path = "updu.conf"
	} else {
		// If path is a directory, append updu.conf
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			path = filepath.Join(path, "updu.conf")
		}
	}

	fmt.Printf("Fetching config from %s...\n", url)
	resp, err := http.Get(url) // #nosec G107
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: fetch failed: %v\n", err)
		osExit(1)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "error: server returned status %d\n", resp.StatusCode)
		osExit(1)
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read failed: %v\n", err)
		osExit(1)
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil { // #nosec G306
		fmt.Fprintf(os.Stderr, "error: write failed: %v\n", err)
		osExit(1)
		return
	}

	fmt.Printf("✓ Config saved to %s\n", path)
}

func handleConfigExport() {
	cfg := config.Load()
	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open database: %v\n", err)
		osExit(1)
		return
	}
	defer db.Close()

	if err := db.Migrate(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to migrate database: %v\n", err)
		osExit(1)
		return
	}

	monitors, err := db.ListMonitors(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to list monitors: %v\n", err)
		osExit(1)
		return
	}

	settings, err := db.ListSettings(context.Background())
	if err != nil {
		settings = make(map[string]string)
	}

	// Add runtime config settings to the map if not in DB
	if _, ok := settings["host"]; !ok {
		settings["host"] = cfg.Host
	}
	if _, ok := settings["port"]; !ok {
		settings["port"] = fmt.Sprintf("%d", cfg.Port)
	}
	if _, ok := settings["base_url"]; !ok {
		settings["base_url"] = cfg.BaseURL
	}
	if _, ok := settings["db_path"]; !ok {
		settings["db_path"] = cfg.DBPath
	}
	if _, ok := settings["log_level"]; !ok {
		settings["log_level"] = cfg.LogLevel
	}

	yCfg := config.FromModels(monitors, settings)
	data, err := yaml.Marshal(yCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to marshal YAML: %v\n", err)
		osExit(1)
		return
	}

	path := "exported.updu.conf"
	if err := os.WriteFile(path, data, 0644); err != nil { // #nosec G306
		fmt.Fprintf(os.Stderr, "error: failed to write file: %v\n", err)
		osExit(1)
		return
	}

	fmt.Printf("✓ Configuration exported to %s\n", path)
}

func serviceInstall() {
	if runtimeGOOS != "linux" {
		fmt.Fprintln(os.Stderr, "error: systemd service install is only supported on Linux")
		osExit(1)
		return
	}

	if osGetuid() != 0 {
		fmt.Fprintln(os.Stderr, "error: must run as root (try sudo)")
		osExit(1)
		return
	}

	// Resolve the absolute path of the current binary
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not resolve executable path: %v\n", err)
		osExit(1)
		return
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not resolve symlinks: %v\n", err)
		osExit(1)
		return
	}

	// Detect working directory (where the binary lives)
	workDir := filepath.Dir(exe)

	// If the binary is in /home or /root, we can't use ProtectHome=true
	protectHome := "true"
	if strings.HasPrefix(workDir, "/home/") || strings.HasPrefix(workDir, "/root/") || workDir == "/root" {
		protectHome = "false"
	}

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
ProtectHome=%s
ReadWritePaths=%s
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`, exe, workDir, protectHome, workDir)

	// Write the unit file
	if err := os.WriteFile(unitPath, []byte(unit), 0644); err != nil { // #nosec G306
		fmt.Fprintf(os.Stderr, "error: could not write unit file: %v\n", err)
		osExit(1)
		return
	}

	fmt.Printf("✓ Wrote %s\n", unitPath)

	// Reload systemd
	if err := systemctl("daemon-reload"); err != nil {
		fmt.Fprintf(os.Stderr, "error: systemctl daemon-reload failed: %v\n", err)
		osExit(1)
		return
	}
	fmt.Println("✓ Reloaded systemd")

	// Enable the service
	if err := systemctl("enable", serviceName); err != nil {
		fmt.Fprintf(os.Stderr, "error: systemctl enable failed: %v\n", err)
		osExit(1)
		return
	}
	fmt.Println("✓ Enabled updu service")

	// Start the service
	if err := systemctl("start", serviceName); err != nil {
		fmt.Fprintf(os.Stderr, "error: systemctl start failed: %v\n", err)
		osExit(1)
		return
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
	if runtimeGOOS != "linux" {
		fmt.Fprintln(os.Stderr, "error: systemd service uninstall is only supported on Linux")
		osExit(1)
		return
	}

	if osGetuid() != 0 {
		fmt.Fprintln(os.Stderr, "error: must run as root (try sudo)")
		osExit(1)
		return
	}

	// Stop the service (ignore errors if not running)
	_ = systemctl("stop", serviceName)
	fmt.Println("✓ Stopped updu service")

	// Disable the service
	_ = systemctl("disable", serviceName)
	fmt.Println("✓ Disabled updu service")

	// Remove the unit file
	if _, err := os.Stat(unitPath); err == nil {
		if err := os.Remove(unitPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: could not remove %s: %v\n", unitPath, err)
			osExit(1)
			return
		}
		fmt.Printf("✓ Removed %s\n", unitPath)
	} else {
		fmt.Println("  (unit file already removed)")
	}

	// Reload systemd
	if err := systemctl("daemon-reload"); err != nil {
		fmt.Fprintf(os.Stderr, "error: systemctl daemon-reload failed: %v\n", err)
		osExit(1)
		return
	}
	fmt.Println("✓ Reloaded systemd")

	fmt.Println()
	fmt.Println("updu service has been uninstalled.")
	fmt.Println("Your data is still in the working directory. Remove it manually if desired.")
}

func handleUpdate(force bool) {
	fmt.Printf("updu %s (%s/%s)\n", version.Version, runtime.GOOS, runtime.GOARCH)
	fmt.Println("Checking for updates...")

	info, err := updater.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		osExit(1)
		return
	}

	if !info.UpdateAvailable && !force {
		fmt.Printf("✓ Already up to date (%s)\n", info.CurrentVersion)
		fmt.Println("Verifying binary checksum...")
		if err := updater.VerifyCurrentBinary(info); err != nil {
			fmt.Fprintf(os.Stderr, "CAUTION: Binary checksum verification failed: %v\n", err)
			fmt.Fprintln(os.Stderr, "This could indicate that the binary has been tampered with.")
			osExit(1)
		}
		fmt.Println("✓ Checksum verified, binary is authentic.")
		osExit(0) // Exit with 0 since it's "success" but no update needed
	}

	if force {
		fmt.Printf("Force update: redownloading %s...\n", info.LatestVersion)
	} else {
		fmt.Printf("Update available: %s → %s\n", info.CurrentVersion, info.LatestVersion)
	}

	if info.AssetURL == "" {
		fmt.Fprintf(os.Stderr, "error: no binary available for %s/%s\n", runtimeGOOS, runtime.GOARCH)
		fmt.Fprintf(os.Stderr, "Download manually from: %s\n", info.ReleaseURL)
		osExit(1)
		return
	}

	fmt.Printf("Downloading %s...\n", info.AssetName)
	if err := updater.DownloadAndApply(info); err != nil {
		fmt.Fprintf(os.Stderr, "error: update failed: %v\n", err)
		osExit(1)
		return
	}

	fmt.Printf("✓ Updated to %s\n", info.LatestVersion)
	fmt.Println()
	fmt.Println("Restart updu to use the new version:")
	fmt.Println("  sudo systemctl restart updu    (if running as a service)")
	fmt.Println("  or simply restart the process")
}

func printUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf(`updu %s - Lightweight Uptime Monitor

Usage:
  %s              Start the updu server
  %s install      Install updu as a systemd service
  %s uninstall    Uninstall the updu systemd service
  %s update       Check for and apply updates from GitHub
  %s update --force Force redownload of the latest binary
  %s version      Show version info
  %s fetch [url]  Download updu.conf from URL (alias for config fetch)
  %s export       Export current config to exported.updu.conf
  %s config fetch [url] Download updu.conf from URL
  %s config export Export current config to exported.updu.conf

Environment Variables:
  UPDU_HOST           Listen address (default: 0.0.0.0)
  UPDU_PORT           Listen port (default: 3000)
  UPDU_DB_PATH        Database file path (default: ./updu.db)
  UPDU_CONF_URL       URL to download updu.conf from (for 'fetch')
  UPDU_CONF_PATH      Dir or file path for updu.conf (default: current dir)
  UPDU_LOG_LEVEL      Log level: debug, info, warn, error (default: info)
  UPDU_ADMIN_USER     Auto-create admin username on first run
  UPDU_ADMIN_PASSWORD  Auto-create admin password on first run

Settings can also be defined in updu.conf (YAML). Environment variables
always have the highest priority. Multiple monitor files can be added
using *.updu.conf files in the same directory as updu.conf.

Systemd Troubleshooting:
  sudo systemctl status updu          Check service status
  sudo systemctl restart updu         Restart the service
  sudo systemctl stop updu            Stop the service
  sudo journalctl -u updu -f          Follow live logs
  sudo journalctl -u updu --since today  Today's logs
  sudo journalctl -u updu -n 50       Last 50 log lines
  systemctl is-active updu            Check if running (for scripts)
  sudo systemctl cat updu             Show the unit file
`, version.Version, exe, exe, exe, exe, exe, exe, exe, exe, exe, exe)
}

func runSystemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...) // #nosec G204
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
	case "update":
		force := false
		if len(os.Args) > 2 && (os.Args[2] == "--force" || os.Args[2] == "-f") {
			force = true
		}
		handleUpdate(force)
		return true
	case "version", "-v", "--version":
		fmt.Printf("updu %s\n", version.Version)
		fmt.Printf("  commit:  %s\n", version.GitCommit)
		fmt.Printf("  built:   %s\n", version.BuildDate)
		fmt.Printf("  go:      %s\n", runtime.Version())
		fmt.Printf("  os/arch: %s/%s\n", runtimeGOOS, runtime.GOARCH)
		return true
	case "help", "-h", "--help":
		printUsage()
		return true
	case "fetch":
		argURL := ""
		if len(os.Args) > 2 {
			argURL = os.Args[2]
		}
		handleConfigFetch(argURL)
		return true
	case "export":
		handleConfigExport()
		return true
	case "config":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "error: 'config' command requires a subcommand (fetch, export)")
			fmt.Printf("Usage: %s config [fetch|export]\n", filepath.Base(os.Args[0]))
			osExit(1)
			return true
		}
		sub := strings.ToLower(os.Args[2])
		if sub == "fetch" {
			argURL := ""
			if len(os.Args) > 3 {
				argURL = os.Args[3]
			}
			handleConfigFetch(argURL)
			return true
		} else if sub == "export" {
			handleConfigExport()
			return true
		}
		fmt.Fprintf(os.Stderr, "error: unknown config subcommand: %s\n", os.Args[2])
		osExit(1)
		return true
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		osExit(1)
		return true
	}
}

func init() {
	slog.Info("updu", "version", version.Version)
}
