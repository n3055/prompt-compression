// Package logger provides structured logging using the standard library's slog package.
package logger

import (
	"log/slog"
	"os"
)

// New creates a structured JSON logger writing to stdout.
func New() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	})
	return slog.New(handler)
}

// NewWithLevel creates a logger with a specific log level.
func NewWithLevel(level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	})
	return slog.New(handler)
}
