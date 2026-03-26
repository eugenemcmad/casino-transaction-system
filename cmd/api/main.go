package main

import (
	"casino-transaction-system/internal/app"
	"casino-transaction-system/internal/config"
	"casino-transaction-system/pkg/logger"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Println(err) // exit 1
		os.Exit(1)
	}

	logger.SetupLogger(cfg.Log.Level)
	slog.Info("Config loaded", "app", cfg.App.Name, "version", cfg.App.Version)

	// Создаем контекст, который отменится при Ctrl+C или docker stop
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	apiApp := app.NewApiApp(cfg)

	// Запуск блокирует main до отмены контекста
	if err := apiApp.Run(ctx); err != nil {
		slog.Error("API stopped with error", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("API gracefully stopped")
}
