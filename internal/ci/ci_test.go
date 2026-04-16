package ci

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type workflow struct {
	Jobs map[string]struct {
		Steps []struct {
			Name string `yaml:"name"`
			Run  string `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func TestCIIncludesFrontendE2EGate(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test path")
	}

	workflowPath := filepath.Join(filepath.Dir(filename), "..", "..", ".github", "workflows", "ci.yml")
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read ci workflow: %v", err)
	}

	var doc workflow
	if err := yaml.Unmarshal(content, &doc); err != nil {
		t.Fatalf("failed to parse ci workflow: %v", err)
	}

	for _, job := range doc.Jobs {
		for _, step := range job.Steps {
			run := strings.ToLower(step.Run)
			if strings.Contains(run, "make e2e-frontend") || strings.Contains(run, "pnpm run test:e2e") {
				return
			}
		}
	}

	t.Fatal("expected CI workflow to run frontend Playwright E2E")
}
