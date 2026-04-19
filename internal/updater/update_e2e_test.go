//go:build e2e

package updater

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSelfUpdateFlow validates the complete self-update lifecycle:
// 1. Build and run an older binary
// 2. Call /api/v1/system/update API
// 3. Verify the binary on disk was replaced with the target release
// 4. Restart and confirm the target version is running
func TestSelfUpdateFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	targetVersion := readTargetVersion(t)
	currentVersion := sourceVersionForUpdate(targetVersion)

	// Setup test workspace
	workDir := t.TempDir()
	oldBinaryPath := filepath.Join(workDir, "updu-old")
	newBinaryPath := filepath.Join(workDir, "updu-new")
	targetBinaryPath := filepath.Join(workDir, "updu")
	dbPath := filepath.Join(workDir, "test.db")

	// Step 1: Build an older binary from the current source tree.
	t.Logf("Building source binary %s...", currentVersion)
	buildBinary(t, oldBinaryPath, currentVersion)

	// Step 2: Build the current release binary that the mock GitHub API will serve.
	t.Logf("Building target binary %s...", targetVersion)
	buildBinary(t, newBinaryPath, targetVersion)

	// Step 3: Copy old binary to target location
	copyFile(t, oldBinaryPath, targetBinaryPath)

	// Step 4: Start mock GitHub API serving the target release.
	githubMock := startMockGitHubAPI(t, newBinaryPath, targetVersion)
	defer githubMock.Close()
	t.Setenv("UPDU_RELEASES_API_URL", githubMock.URL)

	// Step 5: Start the old-version server.
	t.Logf("Starting updu %s server...", currentVersion)
	port := findFreePort(t)
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	var initialLogs bytes.Buffer
	cmd := exec.Command(targetBinaryPath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("UPDU_DB_PATH=%s", dbPath),
		fmt.Sprintf("UPDU_HOST=127.0.0.1"),
		fmt.Sprintf("UPDU_PORT=%d", port),
		"UPDU_AUTH_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"UPDU_LOG_LEVEL=debug",
	)
	cmd.Stdout = &initialLogs
	cmd.Stderr = &initialLogs

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer stopProcess(t, cmd)

	// Wait for server to be ready
	waitForServer(t, serverURL+"/healthz", 10*time.Second)

	// Step 6: Authenticate and get session cookie
	t.Log("Authenticating...")
	sessionCookie := authenticate(t, serverURL, "admin", "password123")
	if strings.Contains(targetVersion, "-") {
		setUpdateChannel(t, serverURL, sessionCookie, ReleaseChannelPrerelease)
	}
	if runningVersion := getRunningVersion(t, serverURL); runningVersion != currentVersion {
		t.Fatalf("expected running version %s before update, got %s", currentVersion, runningVersion)
	}

	// Step 7: Call /api/v1/system/update
	t.Log("Calling /api/v1/system/update...")
	updateResp := callUpdateAPI(t, serverURL, sessionCookie)

	if updateResp["new_version"] != targetVersion {
		t.Fatalf("expected new_version=%s, got %v", targetVersion, updateResp["new_version"])
	}
	if !waitForLogLine(&initialLogs, "checksum verified", 5*time.Second) {
		t.Fatalf("expected updater logs to include checksum verification, got logs:\n%s", initialLogs.String())
	}

	// Step 8: Verify binary was replaced
	t.Log("Verifying binary was replaced...")
	if !verifyBinaryVersion(t, targetBinaryPath, targetVersion) {
		t.Fatalf("binary was not replaced with %s", targetVersion)
	}
	if runningVersion := getRunningVersion(t, serverURL); runningVersion != currentVersion {
		t.Fatalf("expected running process to remain on %s before restart, got %s", currentVersion, runningVersion)
	}

	// Step 9: Stop old process
	stopProcess(t, cmd)

	// Step 10: Restart with new binary
	t.Log("Restarting with new binary...")
	var restartedLogs bytes.Buffer
	cmd = exec.Command(targetBinaryPath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("UPDU_DB_PATH=%s", dbPath),
		fmt.Sprintf("UPDU_HOST=127.0.0.1"),
		fmt.Sprintf("UPDU_PORT=%d", port),
		"UPDU_AUTH_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"UPDU_LOG_LEVEL=debug",
	)
	cmd.Stdout = &restartedLogs
	cmd.Stderr = &restartedLogs

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to restart server: %v", err)
	}
	defer stopProcess(t, cmd)

	// Wait for server to be ready again
	waitForServer(t, serverURL+"/healthz", 10*time.Second)

	// Step 11: Verify new version is running
	t.Log("Verifying new version is running...")
	version := getRunningVersion(t, serverURL)
	if version != targetVersion {
		t.Fatalf("expected running version %s, got %s\ninitial logs:\n%s\nrestarted logs:\n%s", targetVersion, version, initialLogs.String(), restartedLogs.String())
	}
	afterRestartCookie := authenticate(t, serverURL, "admin", "password123")
	versionInfo := callVersionAPI(t, serverURL, afterRestartCookie)
	if versionInfo["current_version"] != targetVersion {
		t.Fatalf("expected current_version=%s, got %v", targetVersion, versionInfo["current_version"])
	}
	if updateAvailable, ok := versionInfo["update_available"].(bool); !ok || updateAvailable {
		t.Fatalf("expected update_available=false, got %v", versionInfo["update_available"])
	}

	t.Log("✓ Self-update flow validated successfully")
}

// buildBinary compiles updu with the specified version
func buildBinary(t *testing.T, outputPath, version string) {
	t.Helper()
	repoRoot := findRepoRoot(t)
	cmd := exec.Command("go", "build",
		"-o", outputPath,
		"-ldflags", fmt.Sprintf("-X github.com/updu/updu/internal/version.Version=%s", version),
		"./cmd/updu",
	)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
}

// findRepoRoot walks up from current directory to find go.mod
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find repository root (go.mod)")
		}
		dir = parent
	}
}

// copyFile copies a file from src to dst
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	input, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}
	if err := os.WriteFile(dst, input, 0755); err != nil {
		t.Fatalf("Failed to write destination file: %v", err)
	}
}

// startMockGitHubAPI starts a test server that returns a mock GitHub release
func startMockGitHubAPI(t *testing.T, binaryPath, version string) *httptest.Server {
	t.Helper()

	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("Failed to read binary for mock: %v", err)
	}
	assetName := platformAssetName()
	checksum := fmt.Sprintf("%x", sha256.Sum256(binaryData))

	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve release list
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, "/releases") {
			releaseJSON := fmt.Sprintf(`[{
				"tag_name": "%s",
				"html_url": "https://github.com/test/updu/releases/%s",
				"body": "Test release notes",
				"published_at": "2026-04-16T00:00:00Z",
				"draft": false,
				"prerelease": false,
				"assets": [{
					"name": "%s",
					"browser_download_url": "%s/download/%s",
					"size": %d
				}]
			}]`, version, version, assetName, ts.URL, assetName, len(binaryData))

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, releaseJSON)
			return
		}

		if r.URL.Path == "/download/checksums.txt" {
			fmt.Fprintf(w, "%s  %s\n", checksum, assetName)
			return
		}

		// Serve binary download
		if r.URL.Path == "/download/"+assetName {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(binaryData)
			return
		}

		http.NotFound(w, r)
	}))

	return ts
}

// findFreePort finds an available TCP port
func findFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// waitForServer polls the health endpoint until it responds or timeout
func waitForServer(t *testing.T, healthURL string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("Server did not become ready within %v", timeout)
}

// authenticate logs in and returns the session cookie
func authenticate(t *testing.T, baseURL, username, password string) string {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	credentials := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)

	registerReq, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/register", strings.NewReader(credentials))
	if err != nil {
		t.Fatalf("failed to create register request: %v", err)
	}
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp, err := client.Do(registerReq)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	registerBody, _ := io.ReadAll(registerResp.Body)
	registerResp.Body.Close()
	if registerResp.StatusCode != http.StatusOK && registerResp.StatusCode != http.StatusForbidden {
		t.Fatalf("register returned %d: %s", registerResp.StatusCode, string(registerBody))
	}

	loginReq, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/login", strings.NewReader(credentials))
	if err != nil {
		t.Fatalf("failed to create login request: %v", err)
	}
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, err := client.Do(loginReq)
	if err != nil {
		t.Fatalf("failed to log in: %v", err)
	}
	loginBody, _ := io.ReadAll(loginResp.Body)
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login returned %d: %s", loginResp.StatusCode, string(loginBody))
	}
	for _, cookie := range loginResp.Cookies() {
		if cookie.Name == "updu_session" {
			return cookie.String()
		}
	}
	t.Fatal("login succeeded but updu_session cookie was not returned")
	return ""
}

// callUpdateAPI calls POST /api/v1/system/update with session auth
func callUpdateAPI(t *testing.T, baseURL, sessionCookie string) map[string]any {
	t.Helper()
	return callJSONAPI(t, http.MethodPost, baseURL+"/api/v1/system/update", sessionCookie, "")
}

// verifyBinaryVersion checks if the binary at path reports the expected version
func verifyBinaryVersion(t *testing.T, binaryPath, expectedVersion string) bool {
	t.Helper()
	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to query binary version: %v\n%s", err, string(output))
	}
	return strings.Contains(string(output), "updu "+expectedVersion)
}

// getRunningVersion queries the running server for its version
func getRunningVersion(t *testing.T, baseURL string) string {
	t.Helper()
	resp, err := http.Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Version endpoint returned %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode version response: %v", err)
	}

	version, ok := result["version"].(string)
	if !ok {
		t.Fatal("Version field not found in response")
	}

	return version
}

func callVersionAPI(t *testing.T, baseURL, sessionCookie string) map[string]any {
	t.Helper()
	return callJSONAPI(t, http.MethodGet, baseURL+"/api/v1/system/version", sessionCookie, "")
}

func setUpdateChannel(t *testing.T, baseURL, sessionCookie, channel string) {
	t.Helper()
	body := fmt.Sprintf(`{"update_channel":%q}`, channel)
	response := callJSONAPI(t, http.MethodPost, baseURL+"/api/v1/settings", sessionCookie, body)
	if response["message"] != "updated" {
		t.Fatalf("expected settings update confirmation, got %v", response)
	}
}

func callJSONAPI(t *testing.T, method, url, sessionCookie, body string) map[string]any {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("failed to create %s request: %v", method, err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if sessionCookie != "" {
		req.Header.Set("Cookie", sessionCookie)
	}

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, url, err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s %s returned %d: %s", method, url, resp.StatusCode, string(responseBody))
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		t.Fatalf("failed to decode %s %s response: %v\n%s", method, url, err, string(responseBody))
	}
	return result
}

func stopProcess(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd == nil || cmd.Process == nil {
		return
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return
	}
	if err := cmd.Process.Kill(); err != nil && !strings.Contains(err.Error(), "process already finished") {
		t.Logf("warning: failed to kill process: %v", err)
	}
	_ = cmd.Wait()
}

func waitForLogLine(logs *bytes.Buffer, substring string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(logs.String(), substring) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return strings.Contains(logs.String(), substring)
}

func readTargetVersion(t *testing.T) string {
	t.Helper()
	repoRoot := findRepoRoot(t)
	cmd := exec.Command("make", "--no-print-directory", "print-version")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to run 'make print-version' in %s: %v", repoRoot, err)
	}
	version := strings.TrimSpace(string(out))
	if version == "" {
		t.Fatal("'make print-version' returned empty string")
	}
	return version
}

func sourceVersionForUpdate(targetVersion string) string {
	if strings.Contains(targetVersion, "-") {
		return "v0.0.1"
	}
	return targetVersion + "-beta"
}
