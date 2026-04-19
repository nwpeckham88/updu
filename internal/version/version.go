// Package version holds build-time version info injected via ldflags.
package version

// These variables are set at build time via:
//
//	go build -ldflags "-X github.com/updu/updu/internal/version.Version=$(git describe --tags --always --dirty)"
//
// In practice, the Makefile derives Version/GitCommit/BuildDate and the
// release workflow overrides Version with the pushed git tag.
var (

	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	BuildTags = "" // e.g. "oidc" — set via ldflags for tagged builds
)
