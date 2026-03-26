package logger

import (
	"log/slog"
	"os"
	"strings"
)

// SetupLogger initializes the default slog logger based on the provided log level.
func SetupLogger(level string) *slog.Logger {
	var slogLevel slog.Level

	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	// Set as default logger
	slog.SetDefault(logger)

	return logger
}
