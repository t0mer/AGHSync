package logging

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

// sensitiveKeyParts are lowercase substrings that mark an attribute key as sensitive.
var sensitiveKeyParts = []string{"password", "token", "secret", "authorization", "credential"}

type redactHandler struct {
	inner slog.Handler
}

func (h *redactHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *redactHandler) Handle(ctx context.Context, r slog.Record) error {
	out := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		out.AddAttrs(redactAttr(a))
		return true
	})
	return h.inner.Handle(ctx, out)
}

func (h *redactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		redacted[i] = redactAttr(a)
	}
	return &redactHandler{inner: h.inner.WithAttrs(redacted)}
}

func (h *redactHandler) WithGroup(name string) slog.Handler {
	return &redactHandler{inner: h.inner.WithGroup(name)}
}

func redactAttr(a slog.Attr) slog.Attr {
	// Recurse into groups first — children may have sensitive keys.
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		redacted := make([]any, 0, len(attrs))
		for _, child := range attrs {
			ra := redactAttr(child)
			redacted = append(redacted, ra.Key, ra.Value.Any())
		}
		return slog.Group(a.Key, redacted...)
	}
	// Check this attribute's own key.
	lower := strings.ToLower(a.Key)
	for _, part := range sensitiveKeyParts {
		if strings.Contains(lower, part) {
			return slog.String(a.Key, "***")
		}
	}
	return a
}

// New returns a JSON slog.Logger at the given level that redacts sensitive attribute values.
func New(level slog.Level, w io.Writer) *slog.Logger {
	inner := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(&redactHandler{inner: inner})
}

// LevelFromString converts a string to a slog.Level, defaulting to LevelWarn for unknown values.
func LevelFromString(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warning", "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}
