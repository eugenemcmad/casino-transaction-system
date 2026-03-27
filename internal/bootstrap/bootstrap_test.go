package bootstrap

import (
	"casino-transaction-system/internal/config"
	"testing"
)

func TestNewApiApp_ReturnsErrorOnRepoInitFailure(t *testing.T) {
	cfg := &config.Config{}
	cfg.Postgres.URL = "postgres://user:pass@127.0.0.1:1/testdb?sslmode=disable"
	cfg.HTTP.Port = "8080"

	app, err := NewApiApp(cfg)
	if err == nil {
		t.Fatal("NewApiApp() expected error, got nil")
	}
	if app != nil {
		t.Fatal("NewApiApp() expected nil app on error")
	}
}

func TestNewProcessorApp_ReturnsErrorOnRepoInitFailure(t *testing.T) {
	cfg := &config.Config{}
	cfg.Postgres.URL = "postgres://user:pass@127.0.0.1:1/testdb?sslmode=disable"
	cfg.Kafka.Brokers = []string{"localhost:9092"}
	cfg.Kafka.Topic = "test-topic"
	cfg.Kafka.GroupID = "test-group"

	app, err := NewProcessorApp(cfg)
	if err == nil {
		t.Fatal("NewProcessorApp() expected error, got nil")
	}
	if app != nil {
		t.Fatal("NewProcessorApp() expected nil app on error")
	}
}
