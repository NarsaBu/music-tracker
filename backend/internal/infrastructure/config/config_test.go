package config

import (
	"strings"
	"testing"
)

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("MB_USER_AGENT", "MusicTracker/0.1 ( test@example.com )")
}

func TestLoad_Defaults(t *testing.T) {
	setRequired(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("HTTP.Addr = %q, want :8080", cfg.HTTP.Addr)
	}
	if cfg.Sync.StaleDays != 90 {
		t.Errorf("Sync.StaleDays = %d, want 90", cfg.Sync.StaleDays)
	}
}

func TestLoad_Validation(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name:    "missing required",
			env:     map[string]string{"DATABASE_URL": "", "MB_USER_AGENT": ""},
			wantErr: "DATABASE_URL",
		},
		{
			name:    "rate limit above MusicBrainz policy",
			env:     map[string]string{"MB_RATE_LIMIT": "2"},
			wantErr: "MB_RATE_LIMIT",
		},
		{
			name:    "unknown log format",
			env:     map[string]string{"LOG_FORMAT": "xml"},
			wantErr: "LOG_FORMAT",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequired(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
