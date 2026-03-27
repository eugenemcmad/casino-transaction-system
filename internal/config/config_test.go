package config

import (
	"os"
	"path/filepath"
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
			t.Setenv("CONFIG_PATH", "")

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

func TestNewConfig_EnvOverridesFileWhenConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	const fileContent = `
app:
  name: file-app
  version: 1.0.0
http:
  port: "8080"
logger:
  log_level: info
postgres:
  pool_max_open: 2
  pool_max_idle: 1
  conn_max_lifetime_minutes: 3
  url: postgres://file:file@localhost:5432/filedb
kafka:
  brokers:
    - localhost:9092
  topic: file-topic
  group_id: file-group
`

	if err := os.WriteFile(configPath, []byte(fileContent), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("CONFIG_PATH", configPath)
	t.Setenv("APP_NAME", "env-app")
	t.Setenv("APP_VERSION", "1.0.0")
	t.Setenv("PG_URL", "postgres://file:file@localhost:5432/filedb")
	t.Setenv("KAFKA_BROKERS", "localhost:9092")
	t.Setenv("KAFKA_TOPIC", "file-topic")
	t.Setenv("KAFKA_GROUP_ID", "env-group")

	ResetConfig()
	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() error = %v", err)
	}

	if cfg.App.Name != "env-app" {
		t.Fatalf("cfg.App.Name = %q, want %q", cfg.App.Name, "env-app")
	}
	if cfg.Kafka.GroupID != "env-group" {
		t.Fatalf("cfg.Kafka.GroupID = %q, want %q", cfg.Kafka.GroupID, "env-group")
	}
}

func TestNewConfig_CanRetryAfterInitialError(t *testing.T) {
	ResetConfig()
	os.Unsetenv("APP_NAME")
	t.Setenv("APP_VERSION", "1.0.0")
	t.Setenv("PG_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("KAFKA_BROKERS", "localhost:9092")
	t.Setenv("KAFKA_TOPIC", "test-topic")
	t.Setenv("CONFIG_PATH", "")

	if _, err := NewConfig(); err == nil {
		t.Fatal("NewConfig() expected initial error, got nil")
	}

	t.Setenv("APP_NAME", "retry-app")
	ResetConfig()

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() retry error = %v", err)
	}
	if cfg.App.Name != "retry-app" {
		t.Fatalf("cfg.App.Name = %q, want %q", cfg.App.Name, "retry-app")
	}
}
