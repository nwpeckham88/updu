package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestSSHChecker(t *testing.T) {
	c := &SSHChecker{}
	if c.Type() != "ssh" {
		t.Error("type")
	}

	if err := c.Validate([]byte(`{"host": "localhost", "port": 22}`)); err != nil {
		t.Error(err)
	}
	if err := c.Validate([]byte(`{"port": 22}`)); err == nil {
		t.Error("expected err")
	}
	if err := c.Validate([]byte(`{bad`)); err == nil {
		t.Error("expected err")
	}

	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)
	monitor := &models.Monitor{
		Config: json.RawMessage(`{"host": "127.0.0.1", "port": 23456}`),
	}
	res, _ := c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Errorf("expected down")
	}

	monitor.Config = json.RawMessage(`{bad`)
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusDown || res.Message == "" {
		t.Errorf("got %v", res.Message)
	}

	// Mock server
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	addr := l.Addr().(*net.TCPAddr)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			conn.Write([]byte("SSH-2.0-OpenSSH_8.9\r\n"))
			conn.Close()
		}
	}()

	monitor.Config = json.RawMessage(fmt.Sprintf(`{"host": "127.0.0.1", "port": %d}`, addr.Port))
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusUp {
		t.Errorf("expected up, got %v msg: %v", res.Status, res.Message)
	}
}
