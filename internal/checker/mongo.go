//go:build mongo

package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/updu/updu/internal/models"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// MongoChecker implements checking for MongoDB clusters/instances.
type MongoChecker struct{}

// Type returns the monitor type
func (c *MongoChecker) Type() string {
	return "mongo"
}

// Check evaluates whether the MongoDB instance is accessible and responding to pings.
func (c *MongoChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: start,
		Status:    models.StatusDown,
	}

	var conf models.MongoMonitorConfig
	if err := json.Unmarshal(monitor.Config, &conf); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	// SSRF protection: extract host from connection string and check
	if parsed, err := url.Parse(conf.ConnectionString); err == nil && parsed.Hostname() != "" {
		if err := CheckHostSSRF(ctx, parsed.Hostname()); err != nil {
			result.Message = err.Error()
			return result, nil
		}
	}

	opts := options.Client().ApplyURI(conf.ConnectionString)
	// Apply the same timeout from our context to the overall client connection logic if possible
	// It's typically better to let the passed ctx handle timeouts for connect & ping

	client, err := mongo.Connect(opts)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to initialize mongo client: %v", err)
		return result, nil
	}
	defer func() {
		// Clean up connection using a background context since current ctx might be done/timed out
		_ = client.Disconnect(context.Background())
	}()

	// Ping the primary
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		result.Message = fmt.Sprintf("Failed to ping MongoDB: %v", err)
		return result, nil
	}

	latency := int(time.Since(start).Milliseconds())
	result.LatencyMs = &latency
	result.Status = models.StatusUp
	result.Message = "MongoDB Ping successful"
	return result, nil
}

// Validate ensures the monitor config is valid for a Mongo Checker
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
