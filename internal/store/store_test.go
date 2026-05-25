package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/store"
)

func TestOpen_CreatesAllTables(t *testing.T) {
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	tables := []string{
		"instances", "sync_config", "sync_runs", "sync_results", "app_config", "schema_migrations",
	}
	for _, table := range tables {
		var name string
		err := s.DB().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		require.NoError(t, err, "table %q should exist", table)
		require.Equal(t, table, name)
	}
}

func TestOpen_MigrationsAreIdempotent(t *testing.T) {
	// Open twice against the same file — second open must not error.
	path := t.TempDir() + "/test.db"

	s1, err := store.Open(path)
	require.NoError(t, err)
	require.NoError(t, s1.Close())

	s2, err := store.Open(path)
	require.NoError(t, err)
	require.NoError(t, s2.Close())
}
