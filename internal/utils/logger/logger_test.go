package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates with default config", func(t *testing.T) {
		l := New(nil)
		assert.NotNil(t, l)
		assert.NotNil(t, l.Logger)
	})

	t.Run("creates with custom config", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cfg := &Config{
			Level:  "debug",
			Format: "json",
			Output: buf,
		}
		l := New(cfg)
		assert.NotNil(t, l)

		l.Info("test message")
		assert.Contains(t, buf.String(), "test message")
	})

	t.Run("creates text format logger", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cfg := &Config{
			Level:  "info",
			Format: "text",
			Output: buf,
		}
		l := New(cfg)

		l.Info("test message")
		output := buf.String()
		assert.Contains(t, output, "test message")
		// Text format should not be JSON
		assert.False(t, strings.HasPrefix(output, "{"))
	})
}

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		level    string
		logFunc  func(*Logger, string)
		expected bool
	}{
		{"debug", func(l *Logger, msg string) { l.Debug(msg) }, true},
		{"info", func(l *Logger, msg string) { l.Info(msg) }, true},
		{"warn", func(l *Logger, msg string) { l.Warn(msg) }, true},
		{"error", func(l *Logger, msg string) { l.Error(msg) }, true},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			buf := &bytes.Buffer{}
			l := New(&Config{
				Level:  tt.level,
				Format: "json",
				Output: buf,
			})

			tt.logFunc(l, "test")
			if tt.expected {
				assert.NotEmpty(t, buf.String())
			}
		})
	}
}

func TestLogger_With(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(&Config{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	l2 := l.With("key", "value")
	l2.Info("test message")

	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "value", entry["key"])
	assert.Equal(t, "test message", entry["msg"])
}

func TestLogger_WithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(&Config{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	l2 := l.WithGroup("mygroup")
	l2.Info("test", "nested", "value")

	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	group, ok := entry["mygroup"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "value", group["nested"])
}

func TestLogger_Context(t *testing.T) {
	t.Run("ContextWithLogger and FromContext", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := New(&Config{
			Level:  "info",
			Format: "json",
			Output: buf,
		})

		ctx := ContextWithLogger(context.Background(), l)
		retrieved := FromContext(ctx)

		assert.Equal(t, l, retrieved)
	})

	t.Run("FromContext returns default when not set", func(t *testing.T) {
		l := FromContext(context.Background())
		assert.NotNil(t, l)
	})
}

func TestLogger_ContextMethods(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(&Config{
		Level:  "debug",
		Format: "json",
		Output: buf,
	})
	ctx := context.Background()

	t.Run("DebugContext", func(t *testing.T) {
		buf.Reset()
		l.DebugContext(ctx, "debug message")
		assert.Contains(t, buf.String(), "debug message")
	})

	t.Run("InfoContext", func(t *testing.T) {
		buf.Reset()
		l.InfoContext(ctx, "info message")
		assert.Contains(t, buf.String(), "info message")
	})

	t.Run("WarnContext", func(t *testing.T) {
		buf.Reset()
		l.WarnContext(ctx, "warn message")
		assert.Contains(t, buf.String(), "warn message")
	})

	t.Run("ErrorContext", func(t *testing.T) {
		buf.Reset()
		l.ErrorContext(ctx, "error message")
		assert.Contains(t, buf.String(), "error message")
	})
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "DEBUG"},
		{"DEBUG", "DEBUG"},
		{"info", "INFO"},
		{"INFO", "INFO"},
		{"warn", "WARN"},
		{"warning", "WARN"},
		{"error", "ERROR"},
		{"ERROR", "ERROR"},
		{"unknown", "INFO"}, // default
		{"", "INFO"},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := parseLevel(tt.input)
			assert.Equal(t, tt.expected, level.String())
		})
	}
}

func TestFieldHelpers(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(&Config{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	l.Info("test",
		String("str", "value"),
		Int("int", 42),
		Int64("int64", 123456789),
		Float64("float", 3.14),
		Bool("bool", true),
		Any("any", map[string]int{"a": 1}),
	)

	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "value", entry["str"])
	assert.Equal(t, float64(42), entry["int"])
	assert.Equal(t, float64(123456789), entry["int64"])
	assert.Equal(t, 3.14, entry["float"])
	assert.Equal(t, true, entry["bool"])
}

func TestErr(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(&Config{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	l.Error("operation failed", Err(assert.AnError))

	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Contains(t, entry["error"], "assert.AnError")
}
