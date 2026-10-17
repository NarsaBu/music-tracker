// Package observability sets up logging.
package observability

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// NewLogger builds a slog.Logger writing to w in the given format ("json" or "text").
func NewLogger(w io.Writer, level, format string) (*slog.Logger, error) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(strings.ToUpper(level))); err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", level, err)
	}
	opts := &slog.HandlerOptions{Level: lvl}

	var h slog.Handler
	switch format {
	case "text":
		h = slog.NewTextHandler(w, opts)
	default:
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(h), nil
}
