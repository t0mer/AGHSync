package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/logging"
)

func TestLogger_RedactsPasswordField(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(slog.LevelDebug, &buf)

	logger.Info("login attempt", "password", "hunter2", "username", "alice")

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "***", rec["password"])
	assert.Equal(t, "alice", rec["username"])
}

func TestLogger_RedactsTokenAndAuthorizationFields(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(slog.LevelDebug, &buf)

	logger.Info("request", "token", "abc", "api_token", "xyz", "authorization", "Basic dXNlcjpwYXNz")

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "***", rec["token"])
	assert.Equal(t, "***", rec["api_token"])
	assert.Equal(t, "***", rec["authorization"])
}

func TestLogger_RespectsLogLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(slog.LevelWarn, &buf)

	logger.Debug("should be dropped")
	logger.Info("should also be dropped")

	assert.Empty(t, buf.String())
}

func TestLevelFromString(t *testing.T) {
	cases := []struct {
		in  string
		out slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warning", slog.LevelWarn},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"UNKNOWN", slog.LevelWarn},
		{"", slog.LevelWarn},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.out, logging.LevelFromString(tc.in), "input: %q", tc.in)
	}
}
