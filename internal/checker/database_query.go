package checker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver

	"github.com/updu/updu/internal/models"
)

// DatabaseQueryChecker executes a database query and validates the result.
type DatabaseQueryChecker struct{}

func (dqc *DatabaseQueryChecker) Type() string {
	return "database_query"
}

func (dqc *DatabaseQueryChecker) Check(ctx context.Context, conf *models.Monitor) (*models.CheckResult, error) {
	result := &models.CheckResult{
		MonitorID: conf.ID,
		Status:    models.StatusDown,
		CheckedAt: time.Now().UTC(),
	}

	var cfg models.DatabaseQueryMonitorConfig
	if err := json.Unmarshal(conf.Config, &cfg); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	if cfg.Engine == "" {
		result.Message = "engine is required"
		return result, nil
	}
	if cfg.Query == "" {
		result.Message = "query is required"
		return result, nil
	}
	if cfg.Comparison == "" {
		cfg.Comparison = "eq"
	}

	// Build connection string based on engine
	var dsn string
	switch cfg.Engine {
	case "postgres":
		if cfg.ConnectionString != "" {
			dsn = cfg.ConnectionString
		} else {
			if cfg.Host == "" {
				cfg.Host = "localhost"
			}
			if cfg.Port == 0 {
				cfg.Port = 5432
			}
			sslMode := cfg.SSLMode
			if sslMode == "" {
				sslMode = "disable"
			}
			dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
				cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, sslMode)
		}
	case "mysql":
		if cfg.ConnectionString != "" {
			dsn = cfg.ConnectionString
		} else {
			if cfg.Host == "" {
				cfg.Host = "localhost"
			}
			if cfg.Port == 0 {
				cfg.Port = 3306
			}
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
				cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		}
	default:
		result.Message = fmt.Sprintf("unsupported engine: %s", cfg.Engine)
		return result, nil
	}

	// Connect with timeout
	start := time.Now()
	db, err := sql.Open(cfg.Engine, dsn)
	if err != nil {
		result.Message = fmt.Sprintf("connect error: %v", err)
		return result, nil
	}
	defer db.Close()

	// Set connection timeout
	dbCtx, cancel := context.WithTimeout(ctx, time.Duration(conf.TimeoutS)*time.Second)
	defer cancel()

	// Test connection
	if err := db.PingContext(dbCtx); err != nil {
		latency := int(time.Since(start).Milliseconds())
		result.LatencyMs = &latency
		result.Message = fmt.Sprintf("ping error: %v", err)
		return result, nil
	}

	// Execute query
	var value string
	if err := db.QueryRowContext(dbCtx, cfg.Query).Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			result.Message = "query returned no rows"
		} else {
			result.Message = fmt.Sprintf("query error: %v", err)
		}
		latency := int(time.Since(start).Milliseconds())
		result.LatencyMs = &latency
		return result, nil
	}

	latency := int(time.Since(start).Milliseconds())
	result.LatencyMs = &latency

	// Compare result
	if compareValues(value, cfg.ExpectedValue, cfg.Comparison) {
		result.Status = models.StatusUp
		result.Message = fmt.Sprintf("query result = %s", value)

		// Capture metadata
		metadata := map[string]interface{}{
			"engine":     cfg.Engine,
			"value":      value,
			"expected":   cfg.ExpectedValue,
			"comparison": cfg.Comparison,
		}
		if data, err := json.Marshal(metadata); err == nil {
			result.Metadata = data
		}
	} else {
		result.Message = fmt.Sprintf("query result = %s (expected %s via %s)", value, cfg.ExpectedValue, cfg.Comparison)
	}

	return result, nil
}

func (dqc *DatabaseQueryChecker) Validate(config json.RawMessage) error {
	var conf models.DatabaseQueryMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid database_query config: %w", err)
	}
	if conf.Engine == "" {
		return fmt.Errorf("engine required")
	}
	valid := map[string]bool{"postgres": true, "mysql": true}
	if !valid[conf.Engine] {
		return fmt.Errorf("unsupported engine: %s", conf.Engine)
	}
	if conf.Query == "" {
		return fmt.Errorf("query required")
	}
	if conf.ExpectedValue == "" {
		return fmt.Errorf("expected_value required")
	}
	validComparisons := map[string]bool{"eq": true, "gt": true, "lt": true, "gte": true, "lte": true}
	if conf.Comparison != "" && !validComparisons[conf.Comparison] {
		return fmt.Errorf("invalid comparison: %s", conf.Comparison)
	}

	// Validate host/port if provided
	if conf.Port != 0 && (conf.Port < 1 || conf.Port > 65535) {
		return fmt.Errorf("port out of range")
	}

	return nil
}
