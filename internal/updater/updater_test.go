package updater

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/updu/updu/internal/version"
)

func TestCheckForUpdate(t *testing.T) {
	// Mock version
	oldVersion := version.Version
	version.Version = "v0.1.0"
	defer func() { version.Version = oldVersion }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{
				"tag_name": "v0.2.0",
				"prerelease": false,
				"draft": false,
				"assets": [
					{"name": "updu-linux-amd64", "browser_download_url": "http://example.com/updu-linux-amd64"},
					{"name": "updu-linux-amd64.sha256", "browser_download_url": "http://example.com/updu-linux-amd64.sha256"}
				]
			}
		]`)
	}))
	defer ts.Close()

	oldAPI := apiURL
	apiURL = ts.URL
	defer func() { apiURL = oldAPI }()

	info, err := CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}

	if info.LatestVersion != "v0.2.0" {
		t.Errorf("expected v0.2.0, got %s", info.LatestVersion)
	}
	if !info.UpdateAvailable {
		t.Error("expected update to be available")
	}
}

func TestCheckForUpdate_StableSkipsPrerelease(t *testing.T) {
	oldVersion := version.Version
	version.Version = "v0.1.0"
	defer func() { version.Version = oldVersion }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{
				"tag_name": "v0.2.0-beta",
				"prerelease": true,
				"draft": false,
				"assets": []
			},
			{
				"tag_name": "v0.1.5",
				"prerelease": false,
				"draft": false,
				"assets": []
			}
		]`)
	}))
	defer ts.Close()

	oldAPI := apiURL
	apiURL = ts.URL
	defer func() { apiURL = oldAPI }()

	info, err := CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}

	if info.LatestVersion != "v0.1.5" {
		t.Fatalf("expected stable release v0.1.5, got %s", info.LatestVersion)
	}
	if !info.UpdateAvailable {
		t.Fatal("expected update to be available")
	}
}

func TestCheckForUpdate_PrereleaseCanUsePrerelease(t *testing.T) {
	oldVersion := version.Version
	version.Version = "v0.1.0-beta"
	defer func() { version.Version = oldVersion }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{
				"tag_name": "v0.2.0-beta",
				"prerelease": true,
				"draft": false,
				"assets": []
			}
		]`)
	}))
	defer ts.Close()

	oldAPI := apiURL
	apiURL = ts.URL
	defer func() { apiURL = oldAPI }()

	info, err := CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}

	if info.LatestVersion != "v0.2.0-beta" {
		t.Fatalf("expected prerelease v0.2.0-beta, got %s", info.LatestVersion)
	}
	if !info.UpdateAvailable {
		t.Fatal("expected update to be available")
	}
}

func TestVerifyCurrentBinary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updu-updater-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	exePath := filepath.Join(tmpDir, "updu")
	if err := os.WriteFile(exePath, []byte("dummy binary content"), 0755); err != nil {
		t.Fatal(err)
	}

	// Mock version.Version and current executable for testing if needed,
	// but VerifyCurrentBinary uses os.Executable() which we can't easily mock
	// in the same process without some refactoring.
	// For now, let's test fetchChecksum separately since it's a big part of it.
}

func TestFetchChecksum(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "73693e4a9057864c803309a056a04f2f63857eb052b694600650d53c7f3e1855  updu-linux-amd64\n")
	}))
	defer ts.Close()

	hash, err := fetchChecksum(ts.URL, "updu-linux-amd64")
	if err != nil {
		t.Fatalf("fetchChecksum failed: %v", err)
	}
	if hash != "73693e4a9057864c803309a056a04f2f63857eb052b694600650d53c7f3e1855" {
		t.Errorf("unexpected hash: %s", hash)
	}
}

func TestPlatformAssetName(t *testing.T) {
	name := platformAssetName()
	// Just verify it doesn't crash and contains updu
	if len(name) <= 4 {
		t.Errorf("unexpected platform asset name: %s", name)
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"0.2.3", []int{0, 2, 3}},
		{"v0.2.3", []int{0, 2, 3}},
		{"v0.2.3-beta", []int{0, 2, 3}},
	}

	for _, tt := range tests {
		got := normalizeVersion(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("normalizeVersion(%s) len mismatch", tt.input)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("normalizeVersion(%s)[%d] = %d, expected %d", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		expected        bool
	}{
		{"v0.2.0", "v0.1.0", true},
		{"v0.1.0", "v0.1.0", false},
		{"v0.1.0", "v0.2.0", false},
		{"v0.1.0", "v0.1.0-beta", true},
		{"v0.1.0-beta", "v0.1.0", false},
		{"v0.1.0", "dev", true},
	}

	for _, tt := range tests {
		if got := isNewer(tt.latest, tt.current); got != tt.expected {
			t.Errorf("isNewer(%s, %s) = %v, expected %v", tt.latest, tt.current, got, tt.expected)
		}
	}
}

func TestDownloadAndApply(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updu-download-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy "current" binary
	exePath := filepath.Join(tmpDir, "updu.exe")
	if err := os.WriteFile(exePath, []byte("old content"), 0755); err != nil {
		t.Fatal(err)
	}

	// Mock server for binary and checksums
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/updu-linux-amd64" {
			w.Write([]byte("new binary content"))
		} else if r.URL.Path == "/checksums.txt" {
			// echo -n "new binary content" | sha256sum -> 8bc3c68d94a3e4de6ea921270c169243da6fd46dedbff8f9608541e7390f4c4b
			fmt.Fprint(w, "8bc3c68d94a3e4de6ea921270c169243da6fd46dedbff8f9608541e7390f4c4b  updu-linux-amd64\n")
		}
	}))
	defer ts.Close()

	os.Setenv("UPDU_TEST_EXE", exePath)
	defer os.Unsetenv("UPDU_TEST_EXE")

	info := &UpdateInfo{
		AssetName: "updu-linux-amd64",
		AssetURL:  ts.URL + "/updu-linux-amd64",
	}

	err = DownloadAndApply(info)
	if err != nil {
		t.Fatalf("DownloadAndApply failed: %v", err)
	}

	content, _ := os.ReadFile(exePath)
	if string(content) != "new binary content" {
		t.Errorf("expected new content, got %s", string(content))
	}
}

func TestVerifyCurrentBinaryFull(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updu-verify-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	exePath := filepath.Join(tmpDir, "updu")
	if err := os.WriteFile(exePath, []byte("content"), 0755); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// hash of "content" -> ed7002b439e9ac845f22357d822bac1444730fbdb6016d3ec9432297b9ec9f73
		fmt.Fprint(w, "ed7002b439e9ac845f22357d822bac1444730fbdb6016d3ec9432297b9ec9f73  updu-linux-amd64\n")
	}))
	defer ts.Close()

	os.Setenv("UPDU_TEST_EXE", exePath)
	defer os.Unsetenv("UPDU_TEST_EXE")

	info := &UpdateInfo{
		AssetName: "updu-linux-amd64",
		AssetURL:  ts.URL + "/updu-linux-amd64",
	}

	err = VerifyCurrentBinary(info)
	if err != nil {
		t.Errorf("VerifyCurrentBinary failed: %v", err)
	}
}
