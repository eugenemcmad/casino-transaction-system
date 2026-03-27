package config

import (
	"os"
	"testing"
)

func TestNewConfig_LoadsAndValidatesEnv(t *testing.T) {
	cases := []struct {
		name        string
		appName     string
		wantErr     bool
		wantAppName string
	}{
		{
			name:        "ok/loads_from_env",
			appName:     "test-app",
			wantErr:     false,
			wantAppName: "test-app",
		},
		{
			name:    "err/returns_error_when_app_name_missing",
			appName: "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.appName == "" {
				os.Unsetenv("APP_NAME")
			} else {
				t.Setenv("APP_NAME", tc.appName)
			}
			t.Setenv("APP_VERSION", "1.0.0")
			t.Setenv("PG_URL", "postgres://user:pass@localhost:5432/db")
			t.Setenv("KAFKA_BROKERS", "localhost:9092")
			t.Setenv("KAFKA_TOPIC", "test-topic")
			os.Unsetenv("CONFIG_PATH")

			ResetConfig()
			cfg, err := NewConfig()
			if (err != nil) != tc.wantErr {
				t.Fatalf("NewConfig() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if cfg.App.Name != tc.wantAppName {
				t.Fatalf("cfg.App.Name = %q, want %q", cfg.App.Name, tc.wantAppName)
			}
			if cfg.Postgres.URL != "postgres://user:pass@localhost:5432/db" {
				t.Fatalf("cfg.Postgres.URL = %q, want %q", cfg.Postgres.URL, "postgres://user:pass@localhost:5432/db")
			}
		})
	}
}
