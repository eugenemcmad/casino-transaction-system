package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestSetupLogger_Levels(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected slog.Level
	}{
		{name: "debug", level: "debug", expected: slog.LevelDebug},
		{name: "info", level: "info", expected: slog.LevelInfo},
		{name: "warn", level: "warn", expected: slog.LevelWarn},
		{name: "error", level: "error", expected: slog.LevelError},
		{name: "default unknown", level: "unknown", expected: slog.LevelInfo},
		{name: "uppercase", level: "DEBUG", expected: slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := SetupLogger(tt.level)
			if l == nil {
				t.Fatal("SetupLogger() returned nil logger")
			}

			if !l.Enabled(context.TODO(), tt.expected) {
				t.Fatalf("logger should be enabled at expected level %v", tt.expected)
			}
			if !l.Handler().Enabled(context.TODO(), tt.expected) {
				t.Fatalf("handler should be enabled at expected level %v", tt.expected)
			}
		})
	}
}
