//go:build !mongo

package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/updu/updu/internal/models"
)

// MongoChecker is a build-time stub used when the binary is compiled without
// the `mongo` build tag. It validates configuration so existing tests and
// GitOps configs still parse, but Check always reports Down with a clear
// message rather than pulling the heavy MongoDB driver into lean builds.
type MongoChecker struct{}

// Type returns the monitor type
func (c *MongoChecker) Type() string {
	return "mongo"
}

// Check reports that MongoDB support is unavailable in this build.
func (c *MongoChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: time.Now(),
		Status:    models.StatusDown,
	}

	var conf models.MongoMonitorConfig
	if err := json.Unmarshal(monitor.Config, &conf); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	result.Message = "MongoDB support not built into this binary (rebuild with -tags mongo)"
	return result, nil
}

// Validate mirrors the real checker so config validation is identical
// regardless of whether MongoDB support is compiled in.
func (c *MongoChecker) Validate(config json.RawMessage) error {
	var conf models.MongoMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid mongo config: %w", err)
	}
	if conf.ConnectionString == "" {
		return fmt.Errorf("connection string is required for mongo monitors")
	}
	return nil
}
