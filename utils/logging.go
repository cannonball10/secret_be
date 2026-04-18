package utils

import (
	"log/slog"
	"os"
)

// InitLogging configures the global slog default logger.
// Production (APP_ENV=production) uses JSON on stderr at Info level.
// All other environments use text on stderr at Debug level.
func InitLogging() {
	var handler slog.Handler
	if os.Getenv("APP_ENV") == "production" {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(handler))
}
