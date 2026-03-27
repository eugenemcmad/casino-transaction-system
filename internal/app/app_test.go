package app

import (
	"casino-transaction-system/internal/config"
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewApiApp_CreatesInstance(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = "8084" // Unique port
	cfg.Postgres.URL = "postgres://localhost:1/db?sslmode=disable"

	_, err := NewApiApp(cfg)
	if err == nil {
		t.Fatal("NewApiApp() expected error for unreachable database, got nil")
	}
}

func TestApiApp_Run_Shutdown(t *testing.T) {
	app := &ApiApp{
		cfg: &config.Config{HTTP: config.HTTP{Port: "0"}},
		server: &http.Server{
			Addr:    ":0",
			Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		},
	}
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
	cfg.Postgres.URL = "postgres://localhost:1/db?sslmode=disable"
	cfg.Kafka.Brokers = []string{"localhost:9092"}
	cfg.Kafka.Topic = "test"
	cfg.Kafka.GroupID = "test-group"

	_, err := NewProcessorApp(cfg)
	if err == nil {
		t.Fatal("NewProcessorApp() expected error for unreachable database, got nil")
	}
}
