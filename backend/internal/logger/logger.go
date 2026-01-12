// Package logger provides structured logging using slog.
package logger

import (
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

// Init initializes the logger with the specified level and format.
// level: "debug", "info", "warn", "error"
// format: "json" or "text"
func Init(level, format string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// Get returns the default logger instance.
func Get() *slog.Logger {
	if defaultLogger == nil {
		// Fallback to default if not initialized
		Init("info", "text")
	}
	return defaultLogger
}

// Info logs an info message with optional key-value pairs.
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Error logs an error message with optional key-value pairs.
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

// Warn logs a warning message with optional key-value pairs.
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Debug logs a debug message with optional key-value pairs.
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}
