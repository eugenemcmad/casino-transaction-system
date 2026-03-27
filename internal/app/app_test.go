package app

import (
	"casino-transaction-system/internal/config"
	"context"
	"net/http"
	"sync"
	"testing"
	"time"
)

type noopCloser struct{}

func (noopCloser) Close() {}

type countingCloser struct {
	mu    sync.Mutex
	calls int
}

func (c *countingCloser) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
}

type noopConsumer struct{}

func (noopConsumer) Start(ctx context.Context) error { return nil }

func TestNewApiApp_CreatesInstance(t *testing.T) {
	cfg := &config.Config{HTTP: config.HTTP{Port: "8084"}}
	server := &http.Server{
		Addr:    ":8084",
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	}

	app := NewApiApp(cfg, server, noopCloser{})
	if app == nil {
		t.Fatal("NewApiApp() returned nil")
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
	cfg.Kafka.Brokers = []string{"localhost:9092"}
	cfg.Kafka.Topic = "test"
	cfg.Kafka.GroupID = "test-group"

	app := NewProcessorApp(cfg, noopConsumer{}, noopCloser{})
	if app == nil {
		t.Fatal("NewProcessorApp() returned nil")
	}
}

func TestApiApp_closeResources_ClosesOnlyOnce(t *testing.T) {
	closer := &countingCloser{}
	app := &ApiApp{closer: closer}

	app.closeResources()
	app.closeResources()

	closer.mu.Lock()
	defer closer.mu.Unlock()
	if closer.calls != 1 {
		t.Fatalf("closeResources() calls = %d, want 1", closer.calls)
	}
}
