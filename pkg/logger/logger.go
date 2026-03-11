package logger

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func init() {
	level := slog.LevelInfo // default: verbose

	if os.Getenv("LOG_QUIET") == "1" {
		level = slog.LevelWarn
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}
