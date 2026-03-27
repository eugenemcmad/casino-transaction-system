// @title Casino Transaction System API
// @version 1.0
// @description API for reading casino transactions.
// @BasePath /
//
// Command api runs the HTTP server for querying transactions (see /swagger/).
package main

import (
	_ "casino-transaction-system/docs"
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

	apiApp, err := bootstrap.NewApiApp(cfg)
	if err != nil {
		slog.Error("Failed to init apiApp", slog.Any("error", err))
		os.Exit(1)
	}

	// Run blocks until the context is cancelled or the server fails.
	if err := apiApp.Run(ctx); err != nil {
		slog.Error("API stopped with error", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("API gracefully stopped")
}
