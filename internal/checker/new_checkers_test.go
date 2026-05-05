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
			name:    "Redis Valid",
			checker: &RedisChecker{},
			config:  `{"host": "localhost", "port": 6379}`,
			wantErr: false,
		},
		{
			name:    "Postgres Valid",
			checker: &PostgresChecker{},
			config:  `{"host": "localhost", "port": 5432}`,
			wantErr: false,
		},
		{
			name:    "MySQL Valid",
			checker: &MySQLChecker{},
			config:  `{"host": "localhost", "port": 3306}`,
			wantErr: false,
		},
		{
			name:    "Mongo Valid",
			checker: &MongoChecker{},
			config:  `{"connection_string": "mongodb://localhost:27017"}`,
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
