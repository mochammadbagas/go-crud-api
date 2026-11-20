package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// ContextHandler wraps a slog.Handler to inject request_id from context automatically.
type ContextHandler struct {
	slog.Handler
}

// NewContextHandler creates a handler that enriches log records with request context.
func NewContextHandler(h slog.Handler) *ContextHandler {
	return &ContextHandler{Handler: h}
}

// Handle extracts request_id from context if available and adds it to the record.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if reqID := chimiddleware.GetReqID(ctx); reqID != "" {
			r.AddAttrs(slog.String("request_id", reqID))
		}
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs returns a new ContextHandler with the given attributes.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup returns a new ContextHandler with the given group.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}

// ParseLevel parses a level string into a slog.Level.
func ParseLevel(lvl string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(lvl)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// New creates and configures a new *slog.Logger according to Config.
func New(cfg *config.Config, out ...io.Writer) *slog.Logger {
	var w io.Writer = os.Stdout
	if len(out) > 0 && out[0] != nil {
		w = out[0]
	}

	level := slog.LevelInfo
	format := "json"
	if cfg != nil {
		level = ParseLevel(cfg.LogLevel)
		format = strings.ToLower(strings.TrimSpace(cfg.LogFormat))
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var baseHandler slog.Handler
	if format == "json" {
		baseHandler = slog.NewJSONHandler(w, opts)
	} else {
		baseHandler = slog.NewTextHandler(w, opts)
	}

	return slog.New(NewContextHandler(baseHandler))
}

// Init configures the default global slog logger using application config.
func Init(cfg *config.Config) *slog.Logger {
	l := New(cfg)
	slog.SetDefault(l)
	return l
}
