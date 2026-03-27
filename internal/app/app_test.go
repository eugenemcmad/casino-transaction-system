package app

import (
	"casino-transaction-system/internal/config"
	"context"
	"testing"
	"time"
)

func TestNewApiApp_CreatesInstance(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = "8084" // Unique port
	cfg.Postgres.URL = "postgres://localhost:5432/db"

	app := NewApiApp(cfg)
	if app == nil {
		t.Fatal("Failed to initialize API app")
	}
}

func TestApiApp_Run_Shutdown(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = "8085"
	cfg.Postgres.URL = "postgres://localhost:5432/db"

	app := NewApiApp(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Testing that Run exits when context is done
	err := app.Run(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Run() returned unexpected error: %v", err)
	}
}

func TestNewProcessorApp_CreatesInstance(t *testing.T) {
	cfg := &config.Config{}
	cfg.Postgres.URL = "postgres://localhost:5432/db"
	cfg.Kafka.Brokers = []string{"localhost:9092"}
	cfg.Kafka.Topic = "test"
	cfg.Kafka.GroupID = "test-group"

	app, err := NewProcessorApp(cfg)
	if err != nil {
		t.Fatalf("NewProcessorApp failed: %v", err)
	}
	if app == nil {
		t.Fatal("Processor app is nil")
	}
}
