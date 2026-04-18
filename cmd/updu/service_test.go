package main

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/config"
)

func TestHandleSubcommandDemoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "custom-demo.conf")

	stdout, stderr, exitCode, handled := runSubcommandForTest(t, []string{"updu", "--demo-config", outPath})

	if !handled {
		t.Fatal("expected --demo-config to be handled as a CLI-only command")
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (stdout=%q stderr=%q)", exitCode, stdout, stderr)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected demo config to be written: %v", err)
	}

	want, err := os.ReadFile(filepath.Join("..", "..", "sample.updu.conf"))
	if err != nil {
		t.Fatalf("failed to read canonical sample config: %v", err)
	}

	if string(got) != string(want) {
		t.Fatal("demo config should match the canonical sample.updu.conf content")
	}

	assertConfigHasAllRegisteredTypes(t, outPath)
	if !strings.Contains(stdout, outPath) {
		t.Fatalf("expected success output to mention %s, got %q", outPath, stdout)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr output, got %q", stderr)
	}
	assertGeneratedConfigFileMode(t, outPath)
}

func TestHandleSubcommandTemplateConfig(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "custom-template.conf")

	stdout, stderr, exitCode, handled := runSubcommandForTest(t, []string{"updu", "--template-config", outPath})

	if !handled {
		t.Fatal("expected --template-config to be handled as a CLI-only command")
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (stdout=%q stderr=%q)", exitCode, stdout, stderr)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected template config to be written: %v", err)
	}

	want, err := os.ReadFile(filepath.Join("..", "..", "examples", "configs", "template", "updu.conf"))
	if err != nil {
		t.Fatalf("failed to read canonical template config: %v", err)
	}

	if string(got) != string(want) {
		t.Fatal("template config should match the canonical template.updu.conf content")
	}

	cfg, err := config.ParseYAMLConfig(outPath)
	if err != nil {
		t.Fatalf("generated template config should parse: %v", err)
	}
	if len(cfg.Monitors) != 0 {
		t.Fatalf("expected template config to keep monitors empty, got %d", len(cfg.Monitors))
	}
	if !strings.Contains(string(got), "# Example monitor types") {
		t.Fatal("template config should include commented examples")
	}
	if !strings.Contains(stdout, outPath) {
		t.Fatalf("expected success output to mention %s, got %q", outPath, stdout)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr output, got %q", stderr)
	}
	assertGeneratedConfigFileMode(t, outPath)
}

func TestHandleSubcommandDemoConfigDefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	stdout, stderr, exitCode, handled := runSubcommandForTest(t, []string{"updu", "--demo-config"})
	if !handled {
		t.Fatal("expected --demo-config to be handled")
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (stdout=%q stderr=%q)", exitCode, stdout, stderr)
	}

	outPath := filepath.Join(tmpDir, "updu-demo.conf")
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected default demo config path to be created: %v", err)
	}
	if !strings.Contains(stdout, "Inspect it first") {
		t.Fatalf("expected default-path guidance in stdout, got %q", stdout)
	}
	if !strings.Contains(stdout, "isolated updu.conf path") {
		t.Fatalf("expected isolated-path guidance in stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr output, got %q", stderr)
	}
	assertGeneratedConfigFileMode(t, outPath)
}

func TestHandleSubcommandTemplateConfigDirectoryTarget(t *testing.T) {
	tmpDir := t.TempDir()

	stdout, stderr, exitCode, handled := runSubcommandForTest(t, []string{"updu", "--template-config", tmpDir})
	if !handled {
		t.Fatal("expected --template-config to be handled")
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (stdout=%q stderr=%q)", exitCode, stdout, stderr)
	}

	outPath := filepath.Join(tmpDir, "updu-template.conf")
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected directory target to resolve to %s: %v", outPath, err)
	}
	if !strings.Contains(stdout, outPath) {
		t.Fatalf("expected success output to mention %s, got %q", outPath, stdout)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr output, got %q", stderr)
	}
	assertGeneratedConfigFileMode(t, outPath)
}

func TestHandleSubcommandGeneratedConfigRefusesOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "existing.conf")
	wantOriginal := "keep-me"
	if err := os.WriteFile(outPath, []byte(wantOriginal), 0600); err != nil {
		t.Fatalf("failed to seed existing file: %v", err)
	}

	stdout, stderr, exitCode, handled := runSubcommandForTest(t, []string{"updu", "--demo-config", outPath})
	if !handled {
		t.Fatal("expected --demo-config to be handled")
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 for overwrite refusal, got %d (stdout=%q stderr=%q)", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected no stdout output on overwrite refusal, got %q", stdout)
	}
	if !strings.Contains(stderr, "already exists") {
		t.Fatalf("expected overwrite refusal on stderr, got %q", stderr)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected original file to remain readable: %v", err)
	}
	if string(got) != wantOriginal {
		t.Fatalf("expected original file contents to remain unchanged, got %q", string(got))
	}
}

func TestHandleSubcommandGeneratedConfigRejectsSymlinkParentPath(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	symlinkParent := filepath.Join(tmpDir, "linked-parent")
	if err := os.Symlink(targetDir, symlinkParent); err != nil {
		t.Skipf("symlink setup not available: %v", err)
	}

	outPath := filepath.Join(symlinkParent, "custom.conf")
	stdout, stderr, exitCode, handled := runSubcommandForTest(t, []string{"updu", "--demo-config", outPath})
	if !handled {
		t.Fatal("expected --demo-config to be handled")
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 for symlink-parent refusal, got %d (stdout=%q stderr=%q)", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected no stdout output, got %q", stdout)
	}
	if stderr == "" {
		t.Fatal("expected stderr output for symlink-parent refusal")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "custom.conf")); !os.IsNotExist(err) {
		t.Fatalf("expected no file to be created through the symlink parent, got err=%v", err)
	}
}

func TestHandleSubcommandGeneratedConfigRejectsInvalidArgs(t *testing.T) {
	testCases := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "too many args",
			args:       []string{"updu", "--demo-config", "one.conf", "two.conf"},
			wantStderr: "accepts at most one optional output path",
		},
		{
			name:       "flag-like path",
			args:       []string{"updu", "--template-config", "--bad"},
			wantStderr: "unexpected flag",
		},
		{
			name:       "auto-loaded fragment path",
			args:       []string{"updu", "--demo-config", "sample.updu.conf"},
			wantStderr: "auto-loaded as companion fragments",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exitCode, handled := runSubcommandForTest(t, tc.args)
			if !handled {
				t.Fatal("expected generated-config command to be handled")
			}
			if exitCode != 1 {
				t.Fatalf("expected exit code 1, got %d (stdout=%q stderr=%q)", exitCode, stdout, stderr)
			}
			if stdout != "" {
				t.Fatalf("expected no stdout output, got %q", stdout)
			}
			if !strings.Contains(stderr, tc.wantStderr) {
				t.Fatalf("expected stderr to contain %q, got %q", tc.wantStderr, stderr)
			}
		})
	}
}

func TestPrintUsageIncludesGeneratedConfigCommands(t *testing.T) {
	stdout, _ := captureOutput(t, func() {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{"updu"}
		printUsage()
	})

	if !strings.Contains(stdout, "--demo-config [path]") {
		t.Fatalf("expected usage to mention --demo-config, got %q", stdout)
	}
	if !strings.Contains(stdout, "--template-config [path]") {
		t.Fatalf("expected usage to mention --template-config, got %q", stdout)
	}
}

func assertConfigHasAllRegisteredTypes(t *testing.T, path string) {
	t.Helper()

	cfg, err := config.ParseYAMLConfig(path)
	if err != nil {
		t.Fatalf("generated config should parse: %v", err)
	}

	presentTypes := make(map[string]struct{}, len(cfg.Monitors))
	for _, monitor := range cfg.Monitors {
		presentTypes[monitor.Type] = struct{}{}
	}

	registry := checker.NewRegistry(false, nil)
	var missingTypes []string
	for _, monitorType := range registry.Types() {
		if _, ok := presentTypes[monitorType]; !ok {
			missingTypes = append(missingTypes, monitorType)
		}
	}
	sort.Strings(missingTypes)
	if len(missingTypes) > 0 {
		t.Fatalf("generated config is missing monitor types: %v", missingTypes)
	}
}

func assertGeneratedConfigFileMode(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat generated config: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected generated config to deny group/other access, got mode %o", info.Mode().Perm())
	}
}

func runSubcommandForTest(t *testing.T, args []string) (string, string, int, bool) {
	t.Helper()

	oldArgs := os.Args
	oldExit := osExit
	defer func() {
		os.Args = oldArgs
		osExit = oldExit
	}()

	exitCode := 0
	handled := false
	os.Args = args
	osExit = func(code int) {
		exitCode = code
	}

	stdout, stderr := captureOutput(t, func() {
		handled = handleSubcommand()
	})

	return stdout, stderr, exitCode, handled
}

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	fn()

	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()

	stdoutBytes, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	stderrBytes, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatalf("failed to read stderr: %v", err)
	}

	return string(stdoutBytes), string(stderrBytes)
}
