package shared

import (
	"log/slog"
	"os"
)

// InitLogger configures the global slog logger.
// verbose enables Debug level (default is Info).
// format is "json" for structured JSON output or "text" for human-readable output.
func InitLogger(verbose bool, format string) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}
