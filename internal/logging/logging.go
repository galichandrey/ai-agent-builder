package logging

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger builds a structured slog.Logger writing JSON to stderr.
// level is parsed case-insensitively: debug, info (default), warn, error.
// Unknown values fall back to info.
func NewLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	})
	return slog.New(handler)
}
