// Package slogx provides structured JSON logging with request ID propagation.
// In production (LOG_FORMAT=json), it outputs JSON lines to stdout.
// In development (default), it outputs human-readable text.
package slogx

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

type contextKey string

const requestIDKey contextKey = "request_id"

var logger *slog.Logger

func init() {
	format := os.Getenv("LOG_FORMAT")
	level := new(slog.LevelVar)

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		level.Set(slog.LevelDebug)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}

	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// WithRequestID returns a context with the given request ID attached.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID extracts the request ID from a context, or returns "unknown".
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok && id != "" {
		return id
	}
	return "unknown"
}

// Logger returns the default structured logger.
func Logger() *slog.Logger {
	return logger
}

func formatMsg(msg string, args ...any) string {
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}

// Info logs at INFO level (printf-style).
func Info(msg string, args ...any) {
	logger.Info(formatMsg(msg, args...))
}

// Warn logs at WARN level (printf-style).
func Warn(msg string, args ...any) {
	logger.Warn(formatMsg(msg, args...))
}

// Error logs at ERROR level (printf-style).
func Error(msg string, args ...any) {
	logger.Error(formatMsg(msg, args...))
}

// Debug logs at DEBUG level (printf-style).
func Debug(msg string, args ...any) {
	logger.Debug(formatMsg(msg, args...))
}

// Fatal logs at ERROR level then exits.
func Fatal(msg string, args ...any) {
	logger.Error(formatMsg(msg, args...))
	os.Exit(1)
}
