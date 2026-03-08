package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestUDPChecker(t *testing.T) {
	c := &UDPChecker{}
	if c.Type() != "udp" {
		t.Errorf("Type() = %v, want udp", c.Type())
	}

	if err := c.Validate([]byte(`{"host": "localhost", "port": 53}`)); err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if err := c.Validate([]byte(`{"port": 53}`)); err == nil {
		t.Error("err expected")
	}
	if err := c.Validate([]byte(`{"host": "localhost"}`)); err == nil {
		t.Error("err expected")
	}
	if err := c.Validate([]byte(`{bad`)); err == nil {
		t.Error("err expected")
	}
	if err := c.Validate([]byte(`{"host": "localhost", "port": 53, "expected_response": "yes"}`)); err == nil {
		t.Error("err expected for expected_resp without payload")
	}

	// Run check tests via custom mock
	l, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	addr := l.LocalAddr().(*net.UDPAddr)

	go func() {
		buf := make([]byte, 1024)
		n, rAddr, _ := l.ReadFrom(buf)
		if n > 0 {
			l.WriteTo([]byte("PONG"), rAddr)
		}
	}()

	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)
	monitor := &models.Monitor{
		ID:     "udp-1",
		Config: json.RawMessage(fmt.Sprintf(`{"host": "127.0.0.1", "port": %d, "send_payload": "PING", "expected_response": "PONG"}`, addr.Port)),
	}

	res, err := c.Check(ctx, monitor)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != models.StatusUp {
		t.Errorf("expected up, got %v - %s", res.Status, res.Message)
	}

	// Without payload
	monitor.Config = json.RawMessage(fmt.Sprintf(`{"host": "127.0.0.1", "port": %d}`, addr.Port))
	res, err = c.Check(ctx, monitor)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != models.StatusUp {
		t.Errorf("expected up (no payload), got %v", res.Status)
	}

	// Bad config
	monitor.Config = json.RawMessage(`{bad`)
	res, _ = c.Check(ctx, monitor)
	if res.Message != "Invalid monitor configuration" {
		t.Errorf("got %v", res.Message)
	}
}
