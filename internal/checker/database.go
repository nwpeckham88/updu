package checker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/updu/updu/internal/models"
)

// DatabaseChecker implements checking for Database connections.
type DatabaseChecker struct{}

// Type returns the monitor type
func (c *DatabaseChecker) Type() string {
	return "database"
}

// Check evaluates whether the Database is accessible and responding to pings.
func (c *DatabaseChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: start,
		Status:    models.StatusDown,
	}

	var conf models.DatabaseMonitorConfig
	if err := json.Unmarshal(monitor.Config, &conf); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	switch conf.Engine {
	case "postgres":
		return c.checkPostgres(ctx, monitor, conf, start, result)
	case "mysql":
		return c.checkMySQL(ctx, monitor, conf, start, result)
	case "redis":
		return c.checkRedis(ctx, monitor, conf, start, result)
	default:
		result.Message = fmt.Sprintf("unsupported engine: %s", conf.Engine)
		return result, nil
	}
}

func (c *DatabaseChecker) checkPostgres(ctx context.Context, monitor *models.Monitor, conf models.DatabaseMonitorConfig, start time.Time, result *models.CheckResult) (*models.CheckResult, error) {
	host := conf.Host
	if host == "" && conf.ConnectionString != "" {
		if u, err := url.Parse(conf.ConnectionString); err == nil && u.Hostname() != "" {
			host = u.Hostname()
		} else {
			for _, part := range strings.Fields(conf.ConnectionString) {
				if strings.HasPrefix(part, "host=") {
					host = strings.TrimPrefix(part, "host=")
					break
				}
			}
		}
	}
	if host != "" {
		if err := CheckHostSSRF(ctx, host); err != nil {
			result.Message = err.Error()
			return result, nil
		}
	}

	dsn := conf.ConnectionString
	if dsn == "" {
		sslMode := conf.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		port := conf.Port
		if port == 0 {
			port = 5432
		}
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			conf.Host, port, conf.User, conf.Password, conf.Database, sslMode)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to initialize postgres driver: %v", err)
		return result, nil
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		result.Message = fmt.Sprintf("Failed to ping PostgreSQL: %v", err)
		return result, nil
	}

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

func (c *DatabaseChecker) checkMySQL(ctx context.Context, monitor *models.Monitor, conf models.DatabaseMonitorConfig, start time.Time, result *models.CheckResult) (*models.CheckResult, error) {
	host := conf.Host
	if host == "" && conf.ConnectionString != "" {
		if u, err := url.Parse(conf.ConnectionString); err == nil && u.Hostname() != "" {
			host = u.Hostname()
		} else if idx := indexTCPHost(conf.ConnectionString); idx != "" {
			host = idx
		}
	}
	if host != "" {
		if err := CheckHostSSRF(ctx, host); err != nil {
			result.Message = err.Error()
			return result, nil
		}
	}

	dsn := conf.ConnectionString
	if dsn == "" {
		auth := conf.User
		if conf.Password != "" {
			auth += ":" + conf.Password
		}
		if auth != "" {
			auth += "@"
		}
		port := conf.Port
		if port == 0 {
			port = 3306
		}
		hostPart := fmt.Sprintf("tcp(%s:%d)", conf.Host, port)
		dsn = fmt.Sprintf("%s%s/%s", auth, hostPart, conf.Database)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to initialize mysql driver: %v", err)
		return result, nil
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		result.Message = fmt.Sprintf("Failed to ping MySQL: %v", err)
		return result, nil
	}

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

func (c *DatabaseChecker) checkRedis(ctx context.Context, monitor *models.Monitor, conf models.DatabaseMonitorConfig, start time.Time, result *models.CheckResult) (*models.CheckResult, error) {
	if err := CheckHostSSRF(ctx, conf.Host); err != nil {
		result.Message = err.Error()
		return result, nil
	}

	port := conf.Port
	if port == 0 {
		port = 6379
	}
	address := fmt.Sprintf("%s:%d", conf.Host, port)

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to connect to Redis: %v", err)
		return result, nil
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	if conf.Password != "" {
		authCmd := fmt.Sprintf("AUTH %s\r\n", conf.Password)
		if _, err := conn.Write([]byte(authCmd)); err != nil {
			result.Message = fmt.Sprintf("Failed to write AUTH command: %v", err)
			return result, nil
		}

		respParams := make([]byte, 1024)
		n, err := conn.Read(respParams)
		if err != nil {
			result.Message = fmt.Sprintf("Empty or failed AUTH response: %v", err)
			return result, nil
		}

		authRes := string(respParams[:n])
		if !strings.HasPrefix(authRes, "+OK") {
			result.Message = fmt.Sprintf("Redis AUTH failed: %s", authRes)
			return result, nil
		}
	}

	// For Redis, conf.Database holds the DB index as a string
	if conf.Database != "" && conf.Database != "0" {
		selectCmd := fmt.Sprintf("SELECT %s\r\n", conf.Database)
		if _, err := conn.Write([]byte(selectCmd)); err != nil {
			result.Message = fmt.Sprintf("Failed to write SELECT command: %v", err)
			return result, nil
		}

		respParams := make([]byte, 1024)
		n, err := conn.Read(respParams)
		if err != nil || !strings.HasPrefix(string(respParams[:n]), "+OK") {
			result.Message = "Redis SELECT DB failed"
			return result, nil
		}
	}

	if _, err := conn.Write([]byte("PING\r\n")); err != nil {
		result.Message = fmt.Sprintf("Failed to write PING command: %v", err)
		return result, nil
	}

	respParams := make([]byte, 1024)
	n, err := conn.Read(respParams)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to read PONG response: %v", err)
		return result, nil
	}

	pongRes := string(respParams[:n])
	if !strings.HasPrefix(pongRes, "+PONG") {
		result.Message = fmt.Sprintf("Unexpected PING response: %s", pongRes)
		return result, nil
	}

	latency := int(time.Since(start).Milliseconds())
	result.LatencyMs = &latency
	result.Status = models.StatusUp
	result.Message = "Redis PING successful"
	return result, nil
}

// Validate ensures the monitor config is valid for a Database Checker
func (c *DatabaseChecker) Validate(config json.RawMessage) error {
	var conf models.DatabaseMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid database config: %w", err)
	}

	if conf.Engine == "" {
		return fmt.Errorf("engine is required for database monitors")
	}

	switch conf.Engine {
	case "postgres":
		if conf.ConnectionString == "" && conf.Host == "" {
			return fmt.Errorf("either connection_string or host is required for postgres monitors")
		}
		if conf.Host != "" && (conf.Port < 0 || conf.Port > 65535) {
			return fmt.Errorf("a valid port (1-65535) is required if host is set (default is 5432)")
		}
	case "mysql":
		if conf.ConnectionString == "" && conf.Host == "" {
			return fmt.Errorf("either connection_string or host is required for mysql monitors")
		}
		if conf.Host != "" && (conf.Port < 0 || conf.Port > 65535) {
			return fmt.Errorf("a valid port (1-65535) is required if host is set (default is 3306)")
		}
	case "redis":
		if conf.Host == "" {
			return fmt.Errorf("host is required for redis monitors")
		}
		if conf.Port < 0 || conf.Port > 65535 {
			return fmt.Errorf("a valid port (1-65535) is required for redis monitors (default is 6379)")
		}
	default:
		return fmt.Errorf("unsupported engine: %s", conf.Engine)
	}

	return nil
}

func indexTCPHost(dsn string) string {
	start := -1
	for _, proto := range []string{"tcp(", "unix("} {
		if i := len(dsn); i > 0 {
			for j := 0; j < len(dsn)-len(proto)+1; j++ {
				if dsn[j:j+len(proto)] == proto {
					start = j + len(proto)
					break
				}
			}
		}
		if start >= 0 {
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := start
	for end < len(dsn) && dsn[end] != ')' && dsn[end] != ':' {
		end++
	}
	return dsn[start:end]
}
