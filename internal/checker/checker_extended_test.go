package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestChecker_Validate_Extended(t *testing.T) {
	checkers := []Checker{
		&UDPChecker{}, &SSHChecker{}, &SMTPChecker{}, &RedisChecker{},
		&PostgresChecker{}, &MySQLChecker{}, &MongoChecker{}, &WebSocketChecker{},
		&PushChecker{}, &JSONAPIChecker{},
	}

	for _, c := range checkers {
		if err := c.Validate(json.RawMessage(`{bad`)); err == nil {
			t.Errorf("expected err for bad json in '%s' checker", c.Type())
		}
	}
}

func TestWebSocketChecker_Real(t *testing.T) {
	c := &WebSocketChecker{}
	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	// 1. Connection Refused
	m := &models.Monitor{
		ID:       "ws-1",
		Config:   json.RawMessage(`{"url":"ws://127.0.0.1:0"}`),
		TimeoutS: 1,
	}
	res, _ := c.Check(ctx, m)
	if res.Status != models.StatusDown {
		t.Errorf("expected Down for WS, got %s", res.Status)
	}

	// 2. Invalid Config
	m.Config = json.RawMessage(`{"url":""}`)
	res, _ = c.Check(ctx, m)
	if res.Status != models.StatusDown {
		t.Errorf("expected Down for WS invalid config, got %s", res.Status)
	}
}

func TestPushChecker_Real(t *testing.T) {
	c := &PushChecker{}
	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	now := time.Now()
	m := &models.Monitor{
		ID:        "push-1",
		Config:    json.RawMessage(`{}`), // Push has no config technically, just a heartbeat ID
		IntervalS: 60,
		TimeoutS:  1,
		LastCheck: &now,
	}
	res, _ := c.Check(ctx, m)
	if res.Status != models.StatusUp {
		t.Errorf("expected Up for Push checker, got %s", res.Status)
	}
}

func TestUDPChecker_Real(t *testing.T) {
	c := &UDPChecker{}
	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	// 1. Bad config (missing host/port)
	m := &models.Monitor{
		ID:       "udp-1",
		Config:   json.RawMessage(`{"send_payload":"ping"}`), // Payload provided but no host -> failure
		TimeoutS: 1,
	}
	res, _ := c.Check(ctx, m)
	if res.Status != models.StatusDown {
		t.Errorf("expected Down for UDP bad config, got %s", res.Status)
	}

	// 2. Local connection failing (UDP is connectionless but we wait for response if payload is provided)
	m.Config = json.RawMessage(`{"host":"127.0.0.1", "port":9999, "send_payload":"ping", "expect_payload":"pong"}`)
	res, _ = c.Check(ctx, m)
	if res.Status != models.StatusDown {
		t.Errorf("expected Down for UDP timeout, got %s", res.Status)
	}
}

func TestSSHChecker_Real(t *testing.T) {
	c := &SSHChecker{}
	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	m := &models.Monitor{
		ID:       "ssh-1",
		Config:   json.RawMessage(`{"host":"127.0.0.1", "port":2}`),
		TimeoutS: 1,
	}
	res, _ := c.Check(ctx, m)
	if res.Status != models.StatusDown {
		t.Errorf("expected Down for SSH refused, got %s", res.Status)
	}
}

func TestSMTPChecker_Real(t *testing.T) {
	c := &SMTPChecker{}
	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	m := &models.Monitor{
		ID:       "smtp-1",
		Config:   json.RawMessage(`{"host":"127.0.0.1", "port":2}`),
		TimeoutS: 1,
	}
	res, _ := c.Check(ctx, m)
	if res.Status != models.StatusDown {
		t.Errorf("expected Down for SMTP refused, got %s", res.Status)
	}
}

func TestDBCheckers_Real(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	checkers := []struct {
		checker Checker
		config  string
	}{
		{&RedisChecker{}, `{"host":"invalid.local.test", "port":9999}`},
		{&PostgresChecker{}, `{"host":"invalid.local.test", "port":9999, "database":"db", "user":"u", "password":"p", "sslmode":"disable"}`},
		{&MySQLChecker{}, `{"host":"invalid.local.test", "port":9999, "database":"db", "user":"u", "password":"p"}`},
		{&MongoChecker{}, `{"connection_string":"mongodb://invalid.local.test:9999"}`},
	}

	for _, tt := range checkers {
		m := &models.Monitor{
			ID:       "db-test",
			Config:   json.RawMessage(tt.config),
			TimeoutS: 1,
		}
		res, _ := tt.checker.Check(ctx, m)
		if res.Status != models.StatusDown {
			t.Errorf("expected Down for %s connection refused, got %s: %s", tt.checker.Type(), res.Status, res.Message)
		}
	}
}

func TestJSONAPIChecker_Real(t *testing.T) {
	c := &JSONAPIChecker{}
	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success", "data":{"value":42}}`))
	}))
	defer ts.Close()

	// Happy path
	m := &models.Monitor{
		ID:       "json-1",
		Config:   json.RawMessage(fmt.Sprintf(`{"url":"%s", "field":"data.value", "expected_value":"42"}`, ts.URL)),
		TimeoutS: 1,
	}
	res, _ := c.Check(ctx, m)
	if res.Status != models.StatusUp {
		t.Errorf("expected Up for JSON API, got %s: %s", res.Status, res.Message)
	}

	// Value mismatch
	m.Config = json.RawMessage(fmt.Sprintf(`{"url":"%s", "field":"data.value", "expected_value":"99"}`, ts.URL))
	res, _ = c.Check(ctx, m)
	if res.Status != models.StatusDown {
		t.Errorf("expected Down for JSON API mismatch, got %s", res.Status)
	}

	// Path not found
	m.Config = json.RawMessage(fmt.Sprintf(`{"url":"%s", "field":"data.missing", "expected_value":"42"}`, ts.URL))
	res, _ = c.Check(ctx, m)
	if res.Status != models.StatusDown || !strings.Contains(res.Message, "not found") {
		t.Errorf("expected Down for JSON API missing path, got %v", res)
	}
}
