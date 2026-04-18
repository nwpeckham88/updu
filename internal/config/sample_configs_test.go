package config

import (
	"path/filepath"
	"testing"

	"github.com/updu/updu/internal/checker"
)

func TestRepositorySampleConfigsParseAndValidate(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		wantMonitors int
	}{
		{
			name:         "root sample",
			path:         filepath.Join("..", "..", "sample.updu.conf"),
			wantMonitors: 19,
		},
		{
			name:         "minimal example",
			path:         filepath.Join("..", "..", "examples", "configs", "minimal", "updu.conf"),
			wantMonitors: 3,
		},
		{
			name:         "homelab example",
			path:         filepath.Join("..", "..", "examples", "configs", "homelab", "updu.conf"),
			wantMonitors: 7,
		},
		{
			name:         "advanced example",
			path:         filepath.Join("..", "..", "examples", "configs", "advanced", "updu.conf"),
			wantMonitors: 6,
		},
		{
			name:         "compose example",
			path:         filepath.Join("..", "..", "examples", "configs", "compose", "updu.conf"),
			wantMonitors: 5,
		},
		{
			name:         "split example",
			path:         filepath.Join("..", "..", "examples", "configs", "split", "updu.conf"),
			wantMonitors: 7,
		},
	}

	registry := checker.NewRegistry(false, nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ParseYAMLConfig(tc.path)
			if err != nil {
				t.Fatalf("ParseYAMLConfig(%s) failed: %v", tc.path, err)
			}

			if got := len(cfg.Monitors); got != tc.wantMonitors {
				t.Fatalf("expected %d monitors, got %d", tc.wantMonitors, got)
			}

			monitors, err := cfg.ToModels()
			if err != nil {
				t.Fatalf("ToModels(%s) failed: %v", tc.path, err)
			}

			for _, monitor := range monitors {
				probe := registry.Get(monitor.Type)
				if probe == nil {
					t.Fatalf("no checker registered for monitor type %q", monitor.Type)
				}

				if err := probe.Validate(monitor.Config); err != nil {
					t.Fatalf("invalid sample monitor %q (%s): %v", monitor.Name, monitor.Type, err)
				}
			}
		})
	}
}
