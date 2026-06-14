package checker

import (
	"testing"
)

func TestExtractExpiryDate(t *testing.T) {
	cases := []struct {
		name     string
		whoisRaw string
		want     string
	}{
		{
			name:     "Registry Expiry Date format",
			whoisRaw: "Domain Name: EXAMPLE.COM\nRegistry Expiry Date: 2025-08-13T04:00:00Z\nRegistrar: Example",
			want:     "2025-08-13T04:00:00Z",
		},
		{
			name:     "Registrar Registration Expiration Date format",
			whoisRaw: "Registrar Registration Expiration Date: 2026-05-15T12:00:00Z",
			want:     "2026-05-15T12:00:00Z",
		},
		{
			name:     "Expiration Date format",
			whoisRaw: "Expiration Date: 02-Jan-2027",
			want:     "02-Jan-2027",
		},
		{
			name:     "Expiry Date format",
			whoisRaw: "Expiry date: 2024-11-22",
			want:     "2024-11-22",
		},
		{
			name:     "paid-till format (.ru)",
			whoisRaw: "state: REGISTERED, DELEGATED, VERIFIED\npaid-till: 2025-10-10T21:00:00Z\nfree-date: 2025-11-11",
			want:     "2025-10-10T21:00:00Z",
		},
		{
			name:     "with trailing junk",
			whoisRaw: "Expiration Date: 2025-01-01 (YYYY-MM-DD)",
			want:     "2025-01-01 (YYYY-MM-DD)", // parsing logic strips this later
		},
		{
			name:     "no match",
			whoisRaw: "Domain Name: NODATE.COM\nStatus: active",
			want:     "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractExpiryDate(tc.whoisRaw)
			if got != tc.want {
				t.Errorf("extractExpiryDate() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	cases := []struct {
		name    string
		dateStr string
		wantYear int
	}{
		{"RFC3339", "2025-08-13T04:00:00Z", 2025},
		{"RFC3339Nano", "2025-08-13T04:00:00.000Z", 2025},
		{"YYYY-MM-DD", "2024-11-22", 2024},
		{"DD-MMM-YYYY", "02-Jan-2027", 2027},
		{"YYYY.MM.DD", "2026.05.15", 2026},
		{"with junk", "2025-01-01 (YYYY-MM-DD)", 2025},
		{"double spaces", "Jan  02 2025", 2025},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDate(tc.dateStr)
			if err != nil {
				t.Fatalf("parseDate(%q) failed: %v", tc.dateStr, err)
			}
			if got.Year() != tc.wantYear {
				t.Errorf("parseDate(%q).Year() = %d, want %d", tc.dateStr, got.Year(), tc.wantYear)
			}
		})
	}
}

func TestGetTLD(t *testing.T) {
	if got := getTLD("example.com"); got != "com" {
		t.Errorf("getTLD(example.com) = %q, want 'com'", got)
	}
	if got := getTLD("sub.example.co.uk"); got != "uk" {
		t.Errorf("getTLD(sub.example.co.uk) = %q, want 'uk'", got)
	}
	if got := getTLD("localhost"); got != "" {
		t.Errorf("getTLD(localhost) = %q, want ''", got)
	}
}
