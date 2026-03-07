package checker

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestChecker_ExhaustiveErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. MySQL Connection Failure
	mysql := &MySQLChecker{}
	res, _ := mysql.Check(ctx, &models.Monitor{
		Config: json.RawMessage(`{"host":"127.0.0.1", "port":3307, "user":"root"}`), // Assuming port 3307 is closed
	})
	if res.Status != models.StatusDown {
		t.Errorf("MySQL: expected Down for closed port, got %s", res.Status)
	}

	// 2. Postgres Connection Failure
	pg := &PostgresChecker{}
	res, _ = pg.Check(ctx, &models.Monitor{
		Config: json.RawMessage(`{"host":"127.0.0.1", "port":5433, "user":"postgres"}`),
	})
	if res.Status != models.StatusDown {
		t.Errorf("Postgres: expected Down for closed port, got %s", res.Status)
	}

	// 3. Mongo Connection Failure
	mongo := &MongoChecker{}
	res, _ = mongo.Check(ctx, &models.Monitor{
		Config: json.RawMessage(`{"host":"127.0.0.1", "port":27018}`),
	})
	if res.Status != models.StatusDown {
		t.Errorf("Mongo: expected Down for closed port, got %s", res.Status)
	}

	// 4. Redis Connection Failure
	redis := &RedisChecker{}
	res, _ = redis.Check(ctx, &models.Monitor{
		Config: json.RawMessage(`{"host":"127.0.0.1", "port":6380}`),
	})
	if res.Status != models.StatusDown {
		t.Errorf("Redis: expected Down for closed port, got %s", res.Status)
	}

	// 5. SMTP Connection Failure
	smtp := &SMTPChecker{}
	res, _ = smtp.Check(ctx, &models.Monitor{
		Config: json.RawMessage(`{"host":"127.0.0.1", "port":2526}`),
	})
	if res.Status != models.StatusDown {
		t.Errorf("SMTP: expected Down for closed port, got %s", res.Status)
	}

	// 6. UDP Connection Failure
	udp := &UDPChecker{}
	res, _ = udp.Check(ctx, &models.Monitor{
		Config: json.RawMessage(`{"host":"127.0.0.1", "port":9999, "send_payload":"ping", "expected_response":"pong"}`),
	})
	if res.Status != models.StatusDown {
		t.Errorf("UDP: expected Down for no response, got %s", res.Status)
	}

	// 7. WebSocket Handshake Failure
	ws := &WebSocketChecker{}
	res, _ = ws.Check(ctx, &models.Monitor{
		Config: json.RawMessage(`{"url":"ws://127.0.0.1:1"}`),
	})
	if res.Status != models.StatusDown {
		t.Errorf("WebSocket: expected Down for closed port, got %s", res.Status)
	}
}

func TestJSONAPIChecker_PathMismatch(t *testing.T) {
	c := &JSONAPIChecker{}
	ctx := context.Background()

	// 1. Path not found
	m := &models.Monitor{
		Config: json.RawMessage(`{"url":"https://api.github.com", "field":"nonexistent.path", "expected_value":"anything"}`),
	}
	res, _ := c.Check(ctx, m)
	if res.Status != models.StatusDown || !strings.Contains(res.Message, "not found") {
		t.Errorf("JSONAPI: expected Down for missing path, got %v", res)
	}

	// 2. Value mismatch
	// Note: We'd need a mock server to be deterministic without network
}

func TestSSHChecker_Failures(t *testing.T) {
	c := &SSHChecker{}
	ctx := context.Background()

	m := &models.Monitor{
		Config: json.RawMessage(`{"host":"127.0.0.1", "port":2223, "user":"root"}`),
	}
	res, _ := c.Check(ctx, m)
	if res.Status != models.StatusDown {
		t.Errorf("SSH: expected Down for closed port, got %s", res.Status)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestChecker_ValidateEdgeCases(t *testing.T) {
	// Test all remaining Validate paths
	configs := []struct {
		c    Checker
		conf string
	}{
		{&MySQLChecker{}, `{"host":""}`},
		{&PostgresChecker{}, `{"host":""}`},
		{&MongoChecker{}, `{"host":""}`},
		{&RedisChecker{}, `{"host":""}`},
		{&SMTPChecker{}, `{"host":""}`},
		{&UDPChecker{}, `{"host":""}`},
		{&WebSocketChecker{}, `{"url":""}`},
		{&JSONAPIChecker{}, `{"url":""}`},
		{&PushChecker{}, `{"slug":""}`},
	}

	for _, tc := range configs {
		if err := tc.c.Validate(json.RawMessage(tc.conf)); err == nil {
			t.Errorf("expected validation error for %s with config %s", tc.c.Type(), tc.conf)
		}
	}
}
