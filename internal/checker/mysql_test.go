package checker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestMySQLChecker(t *testing.T) {
	c := &MySQLChecker{}
	if c.Type() != "mysql" {
		t.Error("type")
	}

	if err := c.Validate([]byte(`{"host": "localhost", "port": 3306}`)); err != nil {
		t.Error(err)
	}
	if err := c.Validate([]byte(`{"port": 3306}`)); err == nil {
		t.Error("expected err")
	}
	if err := c.Validate([]byte(`{"host": "localhost"}`)); err == nil {
		t.Error("expected err")
	}
	if err := c.Validate([]byte(`{"connection_string": "dsn"}`)); err != nil {
		t.Error(err)
	}
	if err := c.Validate([]byte(`{bad`)); err == nil {
		t.Error("expected err")
	}

	ctx := context.Background()
	monitor := &models.Monitor{
		Config: json.RawMessage(`{"host": "127.0.0.1", "port": 23456}`),
	}
	res, _ := c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Error("expected down")
	}

	monitor.Config = json.RawMessage(`{bad`)
	res, _ = c.Check(ctx, monitor)
	if res.Message != "Invalid monitor configuration" {
		t.Errorf("got %v", res.Message)
	}
}
