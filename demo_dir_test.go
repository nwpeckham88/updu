package updu
package updu_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSyncDemoDirCreatesRunnableWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}

	tempDir := t.TempDir()
	binaryDir := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(binaryDir, 0o755); err != nil {
		t.Fatalf("mkdir binary dir: %v", err)
	}

	binaryPath := filepath.Join(binaryDir, "updu")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	configPath := filepath.Join(tempDir, "sample.updu.conf")
	if err := os.WriteFile(configPath, []byte("db_path: ./data/updu.db\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	demoDir := filepath.Join(tempDir, "demo")
	scriptPath := filepath.Join("scripts", "sync-demo-dir.sh")

	run := func() {
		cmd := exec.Command("bash", scriptPath, demoDir)
		cmd.Env = append(os.Environ(),
			"UPDU_BINARY_PATH="+binaryPath,
			"UPDU_DEMO_CONFIG_SOURCE="+configPath,
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("sync demo dir failed: %v\n%s", err, string(output))
		}
	}

	run()
	run()

	if info, err := os.Stat(demoDir); err != nil || !info.IsDir() {
		t.Fatalf("expected demo dir to exist: %v", err)
	}
	if info, err := os.Stat(filepath.Join(demoDir, "data")); err != nil || !info.IsDir() {
		t.Fatalf("expected demo data dir to exist: %v", err)
	}

	binaryLinkPath := filepath.Join(demoDir, "updu")
	binaryLinkTarget, err := os.Readlink(binaryLinkPath)
	if err != nil {
		t.Fatalf("expected demo binary symlink: %v", err)
	}
	if filepath.IsAbs(binaryLinkTarget) {
		t.Fatalf("expected relative binary symlink, got %q", binaryLinkTarget)
	}
	resolvedBinary, err := filepath.EvalSymlinks(binaryLinkPath)
	if err != nil {
		t.Fatalf("resolve binary symlink: %v", err)
	}
	if resolvedBinary != binaryPath {
		t.Fatalf("expected binary symlink to resolve to %q, got %q", binaryPath, resolvedBinary)
	}

	configLinkPath := filepath.Join(demoDir, "updu.conf")
	configLinkTarget, err := os.Readlink(configLinkPath)
	if err != nil {
		t.Fatalf("expected demo config symlink: %v", err)
	}
	if filepath.IsAbs(configLinkTarget) {
		t.Fatalf("expected relative config symlink, got %q", configLinkTarget)
	}
	resolvedConfig, err := filepath.EvalSymlinks(configLinkPath)
	if err != nil {
		t.Fatalf("resolve config symlink: %v", err)
	}
	if resolvedConfig != configPath {
		t.Fatalf("expected config symlink to resolve to %q, got %q", configPath, resolvedConfig)
	}

	content, err := os.ReadFile(configLinkPath)
	if err != nil {
		t.Fatalf("read demo config symlink: %v", err)
	}
	if string(content) != "db_path: ./data/updu.db\n" {
		t.Fatalf("expected demo config content to match source, got %q", string(content))
	}
}