// Package version holds build-time version info injected via ldflags.
package version

// These variables are set at build time via:
//
//	go build -ldflags "-X github.com/updu/updu/internal/version.Version=v0.4.1-beta"
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	BuildTags = "" // e.g. "oidc" — set via ldflags for tagged builds
)
