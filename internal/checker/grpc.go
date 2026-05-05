package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/updu/updu/internal/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// GRPCChecker probes a gRPC server's standard Health service
// (grpc.health.v1.Health/Check) and reports SERVING as Up.
type GRPCChecker struct{}

// Type returns the monitor type.
func (c *GRPCChecker) Type() string {
	return "grpc"
}

// Check evaluates whether the gRPC server is responding with SERVING status.
func (c *GRPCChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: start,
		Status:    models.StatusDown,
	}

	var conf models.GRPCMonitorConfig
	if err := json.Unmarshal(monitor.Config, &conf); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	if err := CheckHostSSRF(ctx, conf.Host); err != nil {
		result.Message = err.Error()
		return result, nil
	}

	target := fmt.Sprintf("%s:%d", conf.Host, conf.Port)

	dialOpts := []grpc.DialOption{}
	if conf.TLS {
		tlsCfg := &tls.Config{
			ServerName:         conf.Host,
			InsecureSkipVerify: conf.InsecureSkipVerify, // #nosec G402 - opt-in via config
			MinVersion:         tls.VersionTLS12,
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if conf.Authority != "" {
		dialOpts = append(dialOpts, grpc.WithAuthority(conf.Authority))
	}

	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to create gRPC client: %v", err)
		return result, nil
	}
	defer conn.Close()

	client := healthpb.NewHealthClient(conn)
	resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{Service: conf.Service})
	if err != nil {
		result.Message = fmt.Sprintf("Health check RPC failed: %v", err)
		return result, nil
	}

	latency := int(time.Since(start).Milliseconds())
	result.LatencyMs = &latency

	if md, mdErr := json.Marshal(map[string]string{
		"service": conf.Service,
		"status":  resp.GetStatus().String(),
	}); mdErr == nil {
		result.Metadata = md
	}

	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		result.Message = fmt.Sprintf("Health status: %s", resp.GetStatus().String())
		return result, nil
	}

	result.Status = models.StatusUp
	result.Message = "gRPC health SERVING"
	return result, nil
}

// Validate ensures the monitor config is valid for a gRPC checker.
func (c *GRPCChecker) Validate(config json.RawMessage) error {
	var conf models.GRPCMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid grpc config: %w", err)
	}
	if conf.Host == "" {
		return fmt.Errorf("host is required for grpc monitors")
	}
	if conf.Port <= 0 || conf.Port > 65535 {
		return fmt.Errorf("a valid port (1-65535) is required for grpc monitors")
	}
	return nil
}
