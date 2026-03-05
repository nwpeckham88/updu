package checker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/updu/updu/internal/models"
)

// MySQLChecker implements checking for MySQL databases.
type MySQLChecker struct{}

// Type returns the monitor type
func (c *MySQLChecker) Type() string {
	return "mysql"
}

// Check evaluates whether the MySQL database is accessible and responding to pings.
func (c *MySQLChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: start,
		Status:    models.StatusDown,
	}

	var conf models.MySQLMonitorConfig
	if err := json.Unmarshal(monitor.Config, &conf); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	dsn := conf.ConnectionString
	if dsn == "" {
		// format user:password@tcp(host:port)/dbname
		auth := conf.User
		if conf.Password != "" {
			auth += ":" + conf.Password
		}
		if auth != "" {
			auth += "@"
		}
		hostPart := fmt.Sprintf("tcp(%s:%d)", conf.Host, conf.Port)
		dsn = fmt.Sprintf("%s%s/%s", auth, hostPart, conf.Database)
	}

	// Open connection (this doesn't actually connect to the database but prepares the driver)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to initialize mysql driver: %v", err)
		return result, nil
	}
	defer db.Close()

	// Use the context for pinging to respect the timeout
	if err := db.PingContext(ctx); err != nil {
		result.Message = fmt.Sprintf("Failed to ping MySQL: %v", err)
		return result, nil
	}

	// MySQL check simple query
	var res int
	err = db.QueryRowContext(ctx, "SELECT 1").Scan(&res)
	if err != nil || res != 1 {
		result.Message = fmt.Sprintf("Failed to execute SELECT 1 (Query check failed): %v", err)
		return result, nil
	}

	latency := int(time.Since(start).Milliseconds())
	result.LatencyMs = &latency
	result.Status = models.StatusUp
	result.Message = "MySQL Ping & Query successful"
	return result, nil
}

// Validate ensures the monitor config is valid for a MySQL Checker
func (c *MySQLChecker) Validate(config json.RawMessage) error {
	var conf models.MySQLMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid mysql config: %w", err)
	}

	if conf.ConnectionString == "" && conf.Host == "" {
		return fmt.Errorf("either connection_string or host is required for mysql monitors")
	}

	if conf.Host != "" && (conf.Port == 0 || conf.Port > 65535) {
		return fmt.Errorf("a valid port (1-65535) is required if host is set (default is 3306)")
	}

	return nil
}
