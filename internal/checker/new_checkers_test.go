package checker

import (
"encoding/json"
"testing"
)

func TestCheckerValidations(t *testing.T) {
tests := []struct {
ame    string
Checker
fig  string
tErr bool
}{
ame:    "Push Valid",
&PushChecker{},
fig:  `{"token": "xyz123"}`,
tErr: false,
ame:    "Push Invalid (No Token)",
&PushChecker{},
fig:  `{}`,
tErr: true,
ame:    "WebSocket Valid",
&WebSocketChecker{},
fig:  `{"url": "wss://echo.websocket.org"}`,
tErr: false,
ame:    "SMTP Valid",
&SMTPChecker{},
fig:  `{"host": "smtp.google.com", "port": 587}`,
tErr: false,
ame:    "UDP Valid",
&UDPChecker{},
fig:  `{"host": "8.8.8.8", "port": 53}`,
tErr: false,
ame:    "Redis Valid",
&RedisChecker{},
fig:  `{"host": "localhost", "port": 6379}`,
tErr: false,
ame:    "Postgres Valid",
&PostgresChecker{},
fig:  `{"host": "localhost", "port": 5432}`,
tErr: false,
ame:    "MySQL Valid",
&MySQLChecker{},
fig:  `{"host": "localhost", "port": 3306}`,
tErr: false,
ame:    "Mongo Valid",
&MongoChecker{},
fig:  `{"connection_string": "mongodb://localhost:27017"}`,
tErr: false,

for _, tt := range tests {
(tt.name, func(t *testing.T) {
:= tt.checker.Validate(json.RawMessage(tt.config))
(err != nil) != tt.wantErr {
error = %v, wantErr %v", err, tt.wantErr)
}
