package checker

import (
	"context"
	"encoding/json"
	"net"
	"os/exec"

	"github.com/updu/updu/internal/models"
)

type contextKey string

const (
	AllowLocalhostKey contextKey = "allow_localhost"
)

// Resolver defines the interface for DNS lookups.
type Resolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
	LookupCNAME(ctx context.Context, host string) (cname string, err error)
	LookupMX(ctx context.Context, host string) ([]*net.MX, error)
	LookupTXT(ctx context.Context, host string) ([]string, error)
	LookupNS(ctx context.Context, host string) ([]*net.NS, error)
}

// Commander defines the interface for executing system commands.
type Commander interface {
	CombinedOutput(ctx context.Context, name string, arg ...string) ([]byte, error)
}

type defaultCommander struct{}

func (c *defaultCommander) CombinedOutput(ctx context.Context, name string, arg ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, arg...).CombinedOutput() // #nosec G204
}

// Checker defines the interface for all monitoring probe types.
type Checker interface {
	// Type returns the monitor type string (e.g., "http", "tcp", "ping").
	Type() string

	// Check performs the actual health check and returns the result.
	Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error)

	// Validate ensures the monitor config is valid for this checker type.
	Validate(config json.RawMessage) error
}

// Registry maps monitor type strings to their Checker implementations.
type Registry struct {
	checkers       map[string]Checker
	AllowLocalhost bool
}

// NewRegistry creates a registry with all built-in checkers.
func NewRegistry(allowLocalhost bool) *Registry {
	r := &Registry{
		checkers:       make(map[string]Checker),
		AllowLocalhost: allowLocalhost,
	}

	// Register built-in checkers
	r.Register(&HTTPChecker{})
	r.Register(&TCPChecker{})
	r.Register(&PingChecker{commander: &defaultCommander{}})
	r.Register(&DNSChecker{resolver: net.DefaultResolver})
	r.Register(&SSLChecker{})
	r.Register(&SSHChecker{})
	r.Register(&JSONAPIChecker{})

	// New general monitors
	r.Register(&PushChecker{})
	r.Register(&WebSocketChecker{})
	r.Register(&SMTPChecker{})
	r.Register(&UDPChecker{})
	r.Register(&RedisChecker{})
	r.Register(&PostgresChecker{})
	r.Register(&MySQLChecker{})
	r.Register(&MongoChecker{})

	return r
}

// Register adds a checker for a given type.
func (r *Registry) Register(c Checker) {
	r.checkers[c.Type()] = c
}

// Get returns the checker for the given type, or nil.
func (r *Registry) Get(typ string) Checker {
	return r.checkers[typ]
}

// Types returns all registered checker type names.
func (r *Registry) Types() []string {
	types := make([]string, 0, len(r.checkers))
	for t := range r.checkers {
		types = append(types, t)
	}
	return types
}
