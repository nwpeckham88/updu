package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseYAMLConfig(t *testing.T) {
	content := `
monitors:
  - id: test-http
    name: Test HTTP
    type: http
    group: Core
    interval: 1m
    config:
      url: https://example.com
  - id: test-tcp
    name: Test TCP
    type: tcp
    groups: ["Infra", "Internal"]
    config:
      host: 1.2.3.4
      port: 80
`
	tmpDir, err := os.MkdirTemp("", "updu-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "updu.yml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseYAMLConfig(configPath)
	if err != nil {
		t.Fatalf("ParseYAMLConfig failed: %v", err)
	}

	if len(cfg.Monitors) != 2 {
		t.Errorf("expected 2 monitors, got %d", len(cfg.Monitors))
	}

	// Check first monitor (legacy 'group' field)
	m1 := cfg.Monitors[0]
	if m1.ID != "test-http" || m1.GroupName != "Core" {
		t.Errorf("m1 mismatch: %+v", m1)
	}

	// Check second monitor (modern 'groups' field)
	m2 := cfg.Monitors[1]
	if m2.ID != "test-tcp" || len(m2.Groups) != 2 || m2.Groups[0] != "Infra" {
		t.Errorf("m2 mismatch: %+v", m2)
	}
}

func TestToModels(t *testing.T) {
	cfg := &YAMLConfig{
		Monitors: []YAMLMonitor{
			{
				ID:        "m1",
				Name:      "M1",
				Type:      "http",
				GroupName: "Legacy",
				Interval:  "30s",
			},
			{
				ID:     "m2",
				Name:   "M2",
				Type:   "tcp",
				Groups: []string{"A", "B"},
			},
		},
	}

	monitors, err := cfg.ToModels()
	if err != nil {
		t.Fatal(err)
	}

	if len(monitors) != 2 {
		t.Fatal("expected 2 monitors")
	}

	if len(monitors[0].Groups) != 1 || monitors[0].Groups[0] != "Legacy" {
		t.Errorf("m0 groups mismatch: %v", monitors[0].Groups)
	}
	if monitors[0].IntervalS != 30 {
		t.Errorf("m0 interval mismatch: %d", monitors[0].IntervalS)
	}

	if len(monitors[1].Groups) != 2 || monitors[1].Groups[0] != "A" {
		t.Errorf("m1 groups mismatch: %v", monitors[1].Groups)
	}
}

func TestParseSimpleDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"30s", 30},
		{"1m", 60},
		{"5m", 300},
		{"1h", 3600},
		{"", 0},        // ToModels handles default
		{"invalid", 0}, // ToModels handles default
	}

	for _, tt := range tests {
		if got := parseSimpleDuration(tt.input); got != tt.expected {
			t.Errorf("parseSimpleDuration(%s) = %d, expected %d", tt.input, got, tt.expected)
		}
	}
}
