package checker

import (
	"encoding/json"
	"testing"
)

func TestCheckerValidations(t *testing.T) {
	tests := []struct {
		name    string
		checker Checker
		config  string
		wantErr bool
	}{
		{
			name:    "Push Valid",
			checker: &PushChecker{},
			config:  `{"token": "xyz123"}`,
			wantErr: false,
		},
		{
			name:    "Push Invalid (No Token)",
			checker: &PushChecker{},
			config:  `{}`,
			wantErr: true,
		},
		{
			name:    "WebSocket Valid",
			checker: &WebSocketChecker{},
			config:  `{"url": "wss://echo.websocket.org"}`,
			wantErr: false,
		},
		{
			name:    "SMTP Valid",
			checker: &SMTPChecker{},
			config:  `{"host": "smtp.google.com", "port": 587}`,
			wantErr: false,
		},
		{
			name:    "UDP Valid",
			checker: &UDPChecker{},
			config:  `{"host": "8.8.8.8", "port": 53}`,
			wantErr: false,
		},
		{
			name:    "Database Redis Valid",
			checker: &DatabaseChecker{},
			config:  `{"engine": "redis", "host": "localhost", "port": 6379}`,
			wantErr: false,
		},
		{
			name:    "Database Postgres Valid",
			checker: &DatabaseChecker{},
			config:  `{"engine": "postgres", "host": "localhost", "port": 5432}`,
			wantErr: false,
		},
		{
			name:    "Database MySQL Valid",
			checker: &DatabaseChecker{},
			config:  `{"engine": "mysql", "host": "localhost", "port": 3306}`,
			wantErr: false,
		},
		{
			name:    "gRPC Valid",
			checker: &GRPCChecker{},
			config:  `{"host": "localhost", "port": 50051}`,
			wantErr: false,
		},
		{
			name:    "gRPC Invalid (no host)",
			checker: &GRPCChecker{},
			config:  `{"port": 50051}`,
			wantErr: true,
		},
		{
			name:    "gRPC Invalid (bad port)",
			checker: &GRPCChecker{},
			config:  `{"host": "localhost", "port": 0}`,
			wantErr: true,
		},
		{
			name:    "Prometheus Valid",
			checker: &PrometheusChecker{},
			config:  `{"host": "localhost", "port": 9090, "metric_name": "up", "expected_value": "1"}`,
			wantErr: false,
		},
		{
			name:    "Prometheus Invalid (no host)",
			checker: &PrometheusChecker{},
			config:  `{"metric_name": "up", "expected_value": "1"}`,
			wantErr: true,
		},
		{
			name:    "Prometheus Invalid (no metric_name)",
			checker: &PrometheusChecker{},
			config:  `{"host": "localhost", "expected_value": "1"}`,
			wantErr: true,
		},
		{
			name:    "DatabaseQuery Valid (postgres)",
			checker: &DatabaseQueryChecker{},
			config:  `{"engine": "postgres", "host": "localhost", "port": 5432, "query": "SELECT 1", "expected_value": "1"}`,
			wantErr: false,
		},
		{
			name:    "DatabaseQuery Invalid (no engine)",
			checker: &DatabaseQueryChecker{},
			config:  `{"host": "localhost", "query": "SELECT 1", "expected_value": "1"}`,
			wantErr: true,
		},
		{
			name:    "DatabaseQuery Invalid (unsupported engine)",
			checker: &DatabaseQueryChecker{},
			config:  `{"engine": "sqlite", "query": "SELECT 1", "expected_value": "1"}`,
			wantErr: true,
		},
		{
			name:    "Sablier Valid",
			checker: &SablierChecker{},
			config:  `{"url": "http://sablier.internal:6660", "service_name": "media"}`,
			wantErr: false,
		},
		{
			name:    "Sablier Invalid (no service_name)",
			checker: &SablierChecker{},
			config:  `{"url": "http://sablier.internal:6660"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.checker.Validate(json.RawMessage(tt.config))
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
