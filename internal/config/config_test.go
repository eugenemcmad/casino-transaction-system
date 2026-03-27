package config

import (
	"os"
	"testing"
)

func TestNewConfig_LoadsFromEnv(t *testing.T) {
	// 1. Setup mock ENV variables
	os.Setenv("APP_NAME", "test-app")
	os.Setenv("APP_VERSION", "1.0.0")
	os.Setenv("PG_URL", "postgres://user:pass@localhost:5432/db")
	os.Setenv("KAFKA_BROKERS", "localhost:9092")
	os.Setenv("KAFKA_TOPIC", "test-topic")

	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("APP_VERSION")
		os.Unsetenv("PG_URL")
		os.Unsetenv("KAFKA_BROKERS")
		os.Unsetenv("KAFKA_TOPIC")
	}()

	// 2. Reset singleton for test
	ResetConfig()

	// 3. Load config
	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 4. Validate
	if cfg.App.Name != "test-app" {
		t.Errorf("Expected APP_NAME test-app, got %s", cfg.App.Name)
	}
	if cfg.Postgres.URL != "postgres://user:pass@localhost:5432/db" {
		t.Errorf("Unexpected PG_URL: %s", cfg.Postgres.URL)
	}
}

func TestNewConfig_ReturnsErrorWhenMandatoryFieldMissing(t *testing.T) {
	// Clear ENV
	os.Unsetenv("APP_NAME")
	ResetConfig()

	_, err := NewConfig()
	if err == nil {
		t.Error("Expected error when mandatory APP_NAME is missing, got nil")
	}
}
