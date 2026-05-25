package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/config"
	"github.com/t0mer/aghsync/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func TestConfig_Defaults(t *testing.T) {
	cfg := config.New(newTestStore(t))

	port, err := cfg.GetPort()
	require.NoError(t, err)
	assert.Equal(t, 8080, port)

	level, err := cfg.GetLogLevel()
	require.NoError(t, err)
	assert.Equal(t, "warning", level)
}

func TestConfig_SetAndGet(t *testing.T) {
	cfg := config.New(newTestStore(t))

	require.NoError(t, cfg.Set("mykey", "myval"))
	val, found, err := cfg.Get("mykey")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "myval", val)
}

func TestConfig_SetEmptyString_NotLostAsDefault(t *testing.T) {
	cfg := config.New(newTestStore(t))

	require.NoError(t, cfg.Set("cron", ""))
	val, err := cfg.GetWithDefault("cron", "fallback")
	require.NoError(t, err)
	assert.Equal(t, "", val, "stored empty string should not be replaced by default")
}

func TestConfig_GetWithDefault_MissingKey(t *testing.T) {
	cfg := config.New(newTestStore(t))

	val, err := cfg.GetWithDefault("nosuchkey", "fallback")
	require.NoError(t, err)
	assert.Equal(t, "fallback", val)
}

func TestConfig_InstallSecret_PersistsAcrossCalls(t *testing.T) {
	cfg := config.New(newTestStore(t))

	s1, err := cfg.InstallSecret()
	require.NoError(t, err)
	assert.Len(t, s1, 32)

	s2, err := cfg.InstallSecret()
	require.NoError(t, err)
	assert.Equal(t, s1, s2, "install secret must be stable across calls")
}

func TestConfig_UIAuth(t *testing.T) {
	s := newTestStore(t)
	cfg := config.New(s)

	// Defaults
	enabled, err := cfg.GetUIAuthEnabled()
	require.NoError(t, err)
	assert.False(t, enabled)

	username, err := cfg.GetUIUsername()
	require.NoError(t, err)
	assert.Equal(t, "", username)

	pwHash, err := cfg.GetUIPasswordHash()
	require.NoError(t, err)
	assert.Equal(t, "", pwHash)

	// Set and retrieve
	require.NoError(t, cfg.SetUIAuthEnabled(true))
	enabled, err = cfg.GetUIAuthEnabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	require.NoError(t, cfg.SetUIUsername("admin"))
	username, err = cfg.GetUIUsername()
	require.NoError(t, err)
	assert.Equal(t, "admin", username)

	require.NoError(t, cfg.SetUIPasswordHash("$2a$12$hash"))
	pwHash, err = cfg.GetUIPasswordHash()
	require.NoError(t, err)
	assert.Equal(t, "$2a$12$hash", pwHash)

	// Disable again
	require.NoError(t, cfg.SetUIAuthEnabled(false))
	enabled, err = cfg.GetUIAuthEnabled()
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestConfig_APIToken(t *testing.T) {
	s := newTestStore(t)
	cfg := config.New(s)

	// Default: no token
	hash, err := cfg.GetAPITokenHash()
	require.NoError(t, err)
	assert.Equal(t, "", hash)

	// Set and retrieve
	require.NoError(t, cfg.SetAPITokenHash("$2a$12$tokenHash"))
	hash, err = cfg.GetAPITokenHash()
	require.NoError(t, err)
	assert.Equal(t, "$2a$12$tokenHash", hash)

	// Clear by passing empty string — should delete the row and return ""
	require.NoError(t, cfg.SetAPITokenHash(""))
	hash, err = cfg.GetAPITokenHash()
	require.NoError(t, err)
	assert.Equal(t, "", hash)
}
