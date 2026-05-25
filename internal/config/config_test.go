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
