package auth

import (
	"testing"

	"github.com/updu/updu/internal/config"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		policy   string
		password string
		wantErr  string
	}{
		{
			name:     "default rejects short passwords",
			policy:   config.PasswordPolicyDefault,
			password: "short",
			wantErr:  "password must be at least 8 characters",
		},
		{
			name:     "off mirrors default minimum length",
			policy:   config.PasswordPolicyOff,
			password: "1234567",
			wantErr:  "password must be at least 8 characters",
		},
		{
			name:     "strong rejects passwords without uppercase letters",
			policy:   config.PasswordPolicyStrong,
			password: "password123",
			wantErr:  "password must be at least 10 characters and include uppercase, lowercase, and a number",
		},
		{
			name:     "strong accepts mixed case alphanumeric passwords",
			policy:   config.PasswordPolicyStrong,
			password: "Password123",
		},
		{
			name:     "very secure rejects passwords without special characters",
			policy:   config.PasswordPolicyVerySecure,
			password: "Password123",
			wantErr:  "password must be at least 12 characters and include uppercase, lowercase, a number, and a special character",
		},
		{
			name:     "very secure accepts fully compliant passwords",
			policy:   config.PasswordPolicyVerySecure,
			password: "Password123!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, tt.policy)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidatePassword() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("ValidatePassword() error = nil, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("ValidatePassword() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}
