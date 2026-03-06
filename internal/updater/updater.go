// Package updater provides self-update functionality using GitHub Releases.
package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/updu/updu/internal/version"
)

const (
	githubOwner = "nwpeckham88"
	githubRepo  = "updu"
	apiURL      = "https://api.github.com/repos/" + githubOwner + "/" + githubRepo + "/releases"
)

// UpdateInfo contains the result of a version check.
type UpdateInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url,omitempty"`
	AssetURL        string `json:"asset_url,omitempty"`
	AssetName       string `json:"asset_name,omitempty"`
	ReleaseNotes    string `json:"release_notes,omitempty"`
	PublishedAt     string `json:"published_at,omitempty"`
}

// githubRelease is the relevant subset of the GitHub API response.
type githubRelease struct {
	TagName     string        `json:"tag_name"`
	HTMLURL     string        `json:"html_url"`
	Body        string        `json:"body"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckForUpdate queries GitHub for the latest release and compares it
// against the current version.
func CheckForUpdate() (*UpdateInfo, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "updu/"+version.Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("contacting GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("GitHub API rate limited (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decoding release: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}
	release := releases[0]

	info := &UpdateInfo{
		CurrentVersion:  version.Version,
		LatestVersion:   release.TagName,
		UpdateAvailable: isNewer(release.TagName, version.Version),
		ReleaseURL:      release.HTMLURL,
		ReleaseNotes:    release.Body,
		PublishedAt:     release.PublishedAt,
	}

	// Find the matching asset for this platform
	assetName := platformAssetName()
	for _, a := range release.Assets {
		if a.Name == assetName {
			info.AssetURL = a.BrowserDownloadURL
			info.AssetName = a.Name
			break
		}
	}

	return info, nil
}

// DownloadAndApply downloads the new binary, verifies its checksum, and
// atomically replaces the current executable.
func DownloadAndApply(info *UpdateInfo) error {
	if info.AssetURL == "" {
		return fmt.Errorf("no download URL for platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	slog.Info("downloading update", "version", info.LatestVersion, "asset", info.AssetName)

	// 1. Find checksums asset
	checksumURL := strings.TrimSuffix(info.AssetURL, info.AssetName) + "checksums.txt"
	expectedHash, err := fetchChecksum(checksumURL, info.AssetName)
	if err != nil {
		slog.Warn("could not fetch checksums, skipping verification", "error", err)
		// Continue without verification — checksums.txt may not exist for all releases.
		expectedHash = ""
	}

	// 2. Download binary to temp file
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(info.AssetURL)
	if err != nil {
		return fmt.Errorf("downloading binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving current executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(exe), "updu-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath) // Clean up on error; no-op if renamed
	}()

	// Write and hash simultaneously
	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)
	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("writing binary: %w", err)
	}
	tmpFile.Close()

	slog.Info("download complete", "bytes", written)

	// 3. Verify checksum
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if expectedHash != "" {
		if actualHash != expectedHash {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
		}
		slog.Info("checksum verified", "sha256", actualHash[:16]+"...")
	}

	// 4. Set executable permissions (match current binary)
	stat, err := os.Stat(exe)
	if err != nil {
		return fmt.Errorf("stat current binary: %w", err)
	}
	if err := os.Chmod(tmpPath, stat.Mode()); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	// 5. Atomic replace: rename temp over current executable
	if err := os.Rename(tmpPath, exe); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	slog.Info("update applied", "version", info.LatestVersion, "path", exe)
	return nil
}

// fetchChecksum downloads checksums.txt and extracts the hash for the given asset.
func fetchChecksum(url, assetName string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums.txt returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == assetName {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("asset %s not found in checksums", assetName)
}

// platformAssetName returns the expected release asset name for the current platform.
func platformAssetName() string {
	switch {
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm":
		return "updu-linux-armv6"
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		return "updu-linux-arm64"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		return "updu-linux-amd64"
	default:
		return fmt.Sprintf("updu-%s-%s", runtime.GOOS, runtime.GOARCH)
	}
}

// isNewer returns true if latest is a higher version than current.
// Handles both "vX.Y.Z" and "vX.Y.Z-beta" formats.
func isNewer(latest, current string) bool {
	// "dev" is always outdated
	if current == "dev" || current == "unknown" {
		return latest != "dev" && latest != "unknown"
	}

	l := normalizeVersion(latest)
	c := normalizeVersion(current)

	// Compare major.minor.patch
	for i := 0; i < 3; i++ {
		lp, cp := 0, 0
		if i < len(l) {
			lp = l[i]
		}
		if i < len(c) {
			cp = c[i]
		}
		if lp > cp {
			return true
		}
		if lp < cp {
			return false
		}
	}

	// If versions are same, a stable release is newer than a pre-release/dirty build.
	return strings.Contains(current, "-") && !strings.Contains(latest, "-")
}

// normalizeVersion strips the "v" prefix and pre-release suffix, returning
// major, minor, patch as ints.
func normalizeVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	// Strip pre-release suffix (e.g., "-beta", "-rc1")
	if idx := strings.IndexByte(v, '-'); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	nums := make([]int, 0, 3)
	for _, p := range parts {
		n := 0
		for _, ch := range p {
			if ch >= '0' && ch <= '9' {
				n = n*10 + int(ch-'0')
			}
		}
		nums = append(nums, n)
	}
	return nums
}
