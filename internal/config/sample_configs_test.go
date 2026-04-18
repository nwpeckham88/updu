package config

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/updu/updu/internal/checker"
)

func TestRepositorySampleConfigsParseAndValidate(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		wantMonitors int
		wantAllTypes bool
	}{
		{
			name:         "root sample",
			path:         filepath.Join("..", "..", "sample.updu.conf"),
			wantMonitors: 19,
			wantAllTypes: true,
		},
		{
			name:         "minimal example",
			path:         filepath.Join("..", "..", "examples", "configs", "minimal", "updu.conf"),
			wantMonitors: 3,
		},
		{
			name:         "template example",
			path:         filepath.Join("..", "..", "examples", "configs", "template", "updu.conf"),
			wantMonitors: 0,
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
			wantMonitors: 19,
			wantAllTypes: true,
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

			if tc.wantAllTypes {
				presentTypes := make(map[string]struct{}, len(cfg.Monitors))
				for _, monitor := range cfg.Monitors {
					presentTypes[monitor.Type] = struct{}{}
				}

				var missingTypes []string
				for _, monitorType := range registry.Types() {
					if _, ok := presentTypes[monitorType]; !ok {
						missingTypes = append(missingTypes, monitorType)
					}
				}
				sort.Strings(missingTypes)
				if len(missingTypes) > 0 {
					t.Fatalf("sample is missing monitor types: %v", missingTypes)
				}
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

func TestTemplateConfigMentionsAllRegisteredTypes(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "configs", "template", "updu.conf")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read template config: %v", err)
	}

	registry := checker.NewRegistry(false, nil)
	mentionedTypes := templateExampleTypes(string(content))
	var missingTypes []string
	for _, monitorType := range registry.Types() {
		if _, ok := mentionedTypes[monitorType]; !ok {
			missingTypes = append(missingTypes, monitorType)
		}
	}
	sort.Strings(missingTypes)
	if len(missingTypes) > 0 {
		t.Fatalf("template config is missing monitor type examples: %v", missingTypes)
	}
}

func templateExampleTypes(content string) map[string]struct{} {
	types := make(map[string]struct{})
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "#") {
			continue
		}

		comment := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if !strings.HasPrefix(comment, "type:") {
			continue
		}

		monitorType := strings.TrimSpace(strings.TrimPrefix(comment, "type:"))
		if monitorType != "" {
			types[monitorType] = struct{}{}
		}
	}

	return types
}
