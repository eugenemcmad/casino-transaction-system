//go:build integration

package bootstrap

import (
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/testutil"
	"testing"
)

func TestNewApiApp_IntegrationSmoke(t *testing.T) {
	connStr, cleanup := testutil.SetupPostgres(t)
	defer cleanup()

	cfg := &config.Config{}
	cfg.Postgres.URL = connStr
	cfg.HTTP.Port = "8089"
	cfg.App.Name = "bootstrap-test"
	cfg.App.Version = "1.0.0"
	cfg.HTTP.ReadHeaderTimeoutSeconds = 5
	cfg.HTTP.ReadTimeoutSeconds = 10
	cfg.HTTP.WriteTimeoutSeconds = 15
	cfg.HTTP.IdleTimeoutSeconds = 60

	app, err := NewApiApp(cfg)
	if err != nil {
		t.Fatalf("NewApiApp() error = %v", err)
	}
	if app == nil {
		t.Fatal("NewApiApp() = nil, want non-nil")
	}
}

func TestNewProcessorApp_IntegrationSmoke(t *testing.T) {
	connStr, pgCleanup := testutil.SetupPostgres(t)
	defer pgCleanup()

	broker, kafkaCleanup := testutil.SetupKafka(t)
	defer kafkaCleanup()

	cfg := &config.Config{}
	cfg.Postgres.URL = connStr
	cfg.Kafka.Brokers = []string{broker}
	cfg.Kafka.Topic = "bootstrap-topic"
	cfg.Kafka.GroupID = "bootstrap-group"
	cfg.App.Name = "bootstrap-test"
	cfg.App.Version = "1.0.0"

	app, err := NewProcessorApp(cfg)
	if err != nil {
		t.Fatalf("NewProcessorApp() error = %v", err)
	}
	if app == nil {
		t.Fatal("NewProcessorApp() = nil, want non-nil")
	}
}
