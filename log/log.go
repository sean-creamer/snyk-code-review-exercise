package log

import (
	"io"
	"log/slog"
	"os"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

type Config struct {
	Out       io.Writer
	Level     slog.Level
	Format    Format
	AddSource bool
}

func New(cfg Config) *slog.Logger {
	if cfg.Out == nil {
		cfg.Out = os.Stdout
	}
	if cfg.Format == "" {
		cfg.Format = FormatJSON
	}

	var handler slog.Handler

	switch cfg.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(cfg.Out, &slog.HandlerOptions{
			Level:     cfg.Level,
			AddSource: cfg.AddSource,
		})
	case FormatText:
		handler = slog.NewTextHandler(cfg.Out, &slog.HandlerOptions{
			Level:     cfg.Level,
			AddSource: cfg.AddSource,
		})
	default:
		panic("invalid log format: must be FormatJSON or FormatText")
	}

	return slog.New(handler)
}
