package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/updu/updu/internal/config"
)

func TestExtractForwardAuth(t *testing.T) {
	cfg := &config.Config{
		ForwardAuthEnabled:     true,
		TrustedProxyCIDRs:      []string{"10.0.0.0/8", "127.0.0.1"},
		ForwardAuthUserHeader:  "Remote-User",
		ForwardAuthEmailHeader: "Remote-Email",
		ForwardAuthGroupHeader: "Remote-Groups",
		ForwardAuthAdminGroup:  "updu-admins",
	}

	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		enabled    bool
		wantNil    bool
		wantUser   string
		wantEmail  string
		wantAdmin  bool
		wantGroups []string
	}{
		{
			name:       "Disabled",
			remoteAddr: "127.0.0.1:12345",
			headers:    map[string]string{"Remote-User": "john"},
			enabled:    false,
			wantNil:    true,
		},
		{
			name:       "Untrusted Proxy",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{"Remote-User": "john"},
			enabled:    true,
			wantNil:    true,
		},
		{
			name:       "Missing User Header",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"Remote-Email": "john@example.com"},
			enabled:    true,
			wantNil:    true,
		},
		{
			name:       "Basic User",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"Remote-User": "john", "Remote-Email": "john@example.com"},
			enabled:    true,
			wantUser:   "john",
			wantEmail:  "john@example.com",
			wantAdmin:  false,
			wantGroups: nil,
		},
		{
			name:       "Admin User",
			remoteAddr: "127.0.0.1",
			headers: map[string]string{
				"Remote-User":   "admin",
				"Remote-Groups": "users, updu-admins, other",
			},
			enabled:    true,
			wantUser:   "admin",
			wantAdmin:  true,
			wantGroups: []string{"users", "updu-admins", "other"},
		},
		{
			name:       "Case Insensitive Admin Group",
			remoteAddr: "10.1.2.3",
			headers: map[string]string{
				"Remote-User":   "admin2",
				"Remote-Groups": "UPDU-ADMINS",
			},
			enabled:    true,
			wantUser:   "admin2",
			wantAdmin:  true,
			wantGroups: []string{"UPDU-ADMINS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.ForwardAuthEnabled = tt.enabled
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			id := ExtractForwardAuth(cfg, req)
			if tt.wantNil {
				if id != nil {
					t.Errorf("Expected nil identity, got %+v", id)
				}
				return
			}
			if id == nil {
				t.Fatalf("Expected identity, got nil")
			}
			if id.Username != tt.wantUser {
				t.Errorf("Username = %q, want %q", id.Username, tt.wantUser)
			}
			if id.Email != tt.wantEmail {
				t.Errorf("Email = %q, want %q", id.Email, tt.wantEmail)
			}
			if id.IsAdmin != tt.wantAdmin {
				t.Errorf("IsAdmin = %v, want %v", id.IsAdmin, tt.wantAdmin)
			}
			if len(id.Groups) != len(tt.wantGroups) {
				t.Errorf("Groups = %v, want %v", id.Groups, tt.wantGroups)
			}
		})
	}
}
