package checker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestMongoChecker(t *testing.T) {
	c := &MongoChecker{}
	if c.Type() != "mongo" {
		t.Error("type")
	}

	if err := c.Validate([]byte(`{"connection_string": "mongodb://localhost:27017"}`)); err != nil {
		t.Error(err)
	}
	if err := c.Validate([]byte(`{}`)); err == nil {
		t.Error("expected err")
	}
	if err := c.Validate([]byte(`{bad`)); err == nil {
		t.Error("expected err")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	monitor := &models.Monitor{
		Config: json.RawMessage(`{"connection_string": "mongodb://user:pass@127.0.0.1:23456"}`),
	}
	res, _ := c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Errorf("expected down, got %v: %v", res.Status, res.Message)
	}

	monitor.Config = json.RawMessage(`{bad`)
	res, _ = c.Check(ctx, monitor)
	if res.Message != "Invalid monitor configuration" {
		t.Errorf("got %v", res.Message)
	}
}
