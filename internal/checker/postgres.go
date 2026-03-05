package checker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/updu/updu/internal/models"
)

// PostgresChecker implements checking for PostgreSQL databases.
type PostgresChecker struct{}

// Type returns the monitor type
func (c *PostgresChecker) Type() string {
	return "postgres"
}

// Check evaluates whether the PostgreSQL database is accessible and responding to pings.
func (c *PostgresChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: start,
		Status:    models.StatusDown,
	}

	var conf models.PostgresMonitorConfig
	if err := json.Unmarshal(monitor.Config, &conf); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	dsn := conf.ConnectionString
	if dsn == "" {
		if conf.SSLMode == "" {
			conf.SSLMode = "disable"
		}
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			conf.Host, conf.Port, conf.User, conf.Password, conf.Database, conf.SSLMode)
	}

	// Open connection (this doesn't actually connect to the database but prepares the driver)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to initialize postgres driver: %v", err)
		return result, nil
	}
	defer db.Close()

	// Use the context for pinging to respect the timeout
	if err := db.PingContext(ctx); err != nil {
		result.Message = fmt.Sprintf("Failed to ping PostgreSQL: %v", err)
		return result, nil
	}

	// Double check we can actually run a simple query
	var res int
	err = db.QueryRowContext(ctx, "SELECT 1").Scan(&res)
	if err != nil || res != 1 {
		result.Message = fmt.Sprintf("Failed to execute SELECT 1 (Query check failed): %v", err)
		return result, nil
	}

	latency := int(time.Since(start).Milliseconds())
	result.LatencyMs = &latency
	result.Status = models.StatusUp
	result.Message = "PostgreSQL Ping & Query successful"
	return result, nil
}

// Validate ensures the monitor config is valid for a Postgres Checker
func (c *PostgresChecker) Validate(config json.RawMessage) error {
	var conf models.PostgresMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid postgres config: %w", err)
	}

	if conf.ConnectionString == "" && conf.Host == "" {
		return fmt.Errorf("either connection_string or host is required for postgres monitors")
	}

	if conf.Host != "" && (conf.Port == 0 || conf.Port > 65535) {
		return fmt.Errorf("a valid port (1-65535) is required if host is set (default is 5432)")
	}

	return nil
}
