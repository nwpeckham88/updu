package checker

import (
	"context"
	"testing"
)

func TestDefaultCommander(t *testing.T) {
	c := &defaultCommander{}
	ctx := context.Background()

	// Test a simple successful command
	out, err := c.CombinedOutput(ctx, "echo", "hello")
	if err != nil {
		t.Fatalf("echo failed: %v", err)
	}
	if string(out) != "hello\n" {
		t.Errorf("expected 'hello\n', got %q", string(out))
	}

	// Test a failing command
	_, err = c.CombinedOutput(ctx, "false")
	if err == nil {
		t.Error("expected error for 'false' command, got nil")
	}
}
