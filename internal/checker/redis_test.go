package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestRedisChecker(t *testing.T) {
	c := &RedisChecker{}
	if c.Type() != "redis" {
		t.Errorf("Type() = %v, want redis", c.Type())
	}

	if err := c.Validate([]byte(`{"host": "localhost", "port": 6379}`)); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate([]byte(`{"port": 6379}`)); err == nil {
		t.Error("expected err")
	}
	if err := c.Validate([]byte(`{"host": "localhost"}`)); err == nil {
		t.Error("expected err")
	}
	if err := c.Validate([]byte(`{bad`)); err == nil {
		t.Error("expected err")
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

			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if err != nil || n == 0 {
						return
					}
					req := string(buf[:n])
					if req == "PING\r\n" {
						c.Write([]byte("+PONG\r\n"))
					} else {
						c.Write([]byte("+OK\r\n"))
					}
				}
			}(conn)
		}
	}()

	ctx := context.Background()
	monitor := &models.Monitor{
		ID:     "redis-1",
		Config: json.RawMessage(fmt.Sprintf(`{"host": "127.0.0.1", "port": %d}`, addr.Port)),
	}

	res, _ := c.Check(ctx, monitor)
	if res.Status != models.StatusUp {
		t.Errorf("expected up, got %v", res.Status)
	}

	monitor.Config = json.RawMessage(fmt.Sprintf(`{"host": "127.0.0.1", "port": %d, "password": "pass", "database": 1}`, addr.Port))
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusUp {
		t.Errorf("expected up, got %v msg: %v", res.Status, res.Message)
	}

	// Bad config
	monitor.Config = json.RawMessage(`{bad`)
	res, _ = c.Check(ctx, monitor)
	if res.Message != "Invalid monitor configuration" {
		t.Errorf("got %v", res.Message)
	}
}
