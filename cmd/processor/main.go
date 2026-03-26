// Command processor runs the Kafka consumer that persists transactions to PostgreSQL.
package main

import (
	"casino-transaction-system/internal/bootstrap"
	"casino-transaction-system/internal/config"
	"casino-transaction-system/pkg/logger"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	logger.SetupLogger(cfg.Log.Level)
	slog.Info("Config loaded", "app", cfg.App.Name, "version", cfg.App.Version)

	// Cancel on SIGINT/SIGTERM (e.g. Ctrl+C or docker stop).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	processorApp, err := bootstrap.NewProcessorApp(cfg)
	if err != nil {
		slog.Error("Failed to init processorApp", slog.Any("error", err))
		os.Exit(1)
	}

	// Run blocks until the context is cancelled or the consumer returns an error.
	if err := processorApp.Run(ctx); err != nil {
		slog.Error("Processor stopped with error", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Processor gracefully stopped")
}
