package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestSMTPChecker(t *testing.T) {
	c := &SMTPChecker{}
	if c.Type() != "smtp" {
		t.Errorf("Type() = %v, want smtp", c.Type())
	}

	if err := c.Validate([]byte(`{"host": "localhost", "port": 25}`)); err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if err := c.Validate([]byte(`{"port": 25}`)); err == nil {
		t.Errorf("Expected error for no host")
	}
	if err := c.Validate([]byte(`{"host": "localhost"}`)); err == nil {
		t.Errorf("Expected error for no port")
	}
	if err := c.Validate([]byte(`{bad`)); err == nil {
		t.Errorf("Expected error for bad json")
	}

	monitor := &models.Monitor{
		ID:     "smtp-1",
		Config: json.RawMessage(`{bad`),
	}
	ctx := context.Background()
	res, err := c.Check(ctx, monitor)
	if err != nil {
		t.Errorf("Expected nil err, got %v", err)
	}
	if res.Message != "Invalid monitor configuration" {
		t.Errorf("Expected config err msg, got %v", res.Message)
	}

	monitor.Config = json.RawMessage(`{"host": "127.0.0.1", "port": 12345}`)
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Errorf("Expected StatusDown, got %v", res.Status)
	}

	// Start a mock TCP server
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.Write([]byte("220 mock.smtp SMTP\r\n"))
		buf := make([]byte, 1024)
		conn.Read(buf)
		conn.Write([]byte("250 OK\r\n"))
	}()

	addr := l.Addr().(*net.TCPAddr)
	monitor.Config = json.RawMessage(fmt.Sprintf(`{"host": "127.0.0.1", "port": %d}`, addr.Port))
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusUp {
		t.Errorf("Expected StatusUp, got %v - %s", res.Status, res.Message)
	}
}
