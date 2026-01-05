package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger with additional convenience methods.
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration.
type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, text
	Output io.Writer
}

// DefaultConfig returns default logger configuration.
func DefaultConfig() *Config {
	return &Config{
		Level:  "info",
		Format: "json",
		Output: os.Stdout,
	}
}

// New creates a new Logger with the given configuration.
func New(cfg *Config) *Logger {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "text":
		handler = slog.NewTextHandler(cfg.Output, opts)
	default:
		handler = slog.NewJSONHandler(cfg.Output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// parseLevel parses a log level string.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// With returns a new Logger with the given attributes.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}

// WithGroup returns a new Logger with the given group name.
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		Logger: l.Logger.WithGroup(name),
	}
}

// --- Context-aware logging ---

type contextKey struct{}

// ContextWithLogger returns a new context with the logger.
func ContextWithLogger(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext returns the logger from context, or a default logger.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(contextKey{}).(*Logger); ok {
		return l
	}
	return New(nil)
}

// --- Convenience methods with context ---

// DebugContext logs at debug level with context.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.Logger.DebugContext(ctx, msg, args...)
}

// InfoContext logs at info level with context.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.Logger.InfoContext(ctx, msg, args...)
}

// WarnContext logs at warn level with context.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.Logger.WarnContext(ctx, msg, args...)
}

// ErrorContext logs at error level with context.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.Logger.ErrorContext(ctx, msg, args...)
}

// --- Common field helpers ---

// String returns a string attribute.
func String(key, value string) slog.Attr {
	return slog.String(key, value)
}

// Int returns an int attribute.
func Int(key string, value int) slog.Attr {
	return slog.Int(key, value)
}

// Int64 returns an int64 attribute.
func Int64(key string, value int64) slog.Attr {
	return slog.Int64(key, value)
}

// Float64 returns a float64 attribute.
func Float64(key string, value float64) slog.Attr {
	return slog.Float64(key, value)
}

// Bool returns a bool attribute.
func Bool(key string, value bool) slog.Attr {
	return slog.Bool(key, value)
}

// Any returns an attribute for any value.
func Any(key string, value any) slog.Attr {
	return slog.Any(key, value)
}

// Err returns an error attribute.
func Err(err error) slog.Attr {
	return slog.Any("error", err)
}

// Duration returns a duration attribute.
func Duration(key string, d any) slog.Attr {
	return slog.Any(key, d)
}

// Group returns a group attribute.
func Group(key string, args ...any) slog.Attr {
	return slog.Group(key, args...)
}
