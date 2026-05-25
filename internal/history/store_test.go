package history_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/history"
	"github.com/t0mer/aghsync/internal/store"
)

func newTestStore(t *testing.T) *history.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return history.New(s.DB())
}

// insertTestInstance inserts a minimal instance row so FK constraints on sync_results are satisfied.
func insertTestInstance(t *testing.T, s *history.Store, id string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB().Exec(
		`INSERT INTO instances(id, name, address, username, password_enc, is_master, tls_skip_verify, created_at, updated_at)
		 VALUES(?,?,?,?,?,0,0,?,?)`,
		id, id, "http://localhost", "", "", now, now,
	)
	require.NoError(t, err)
}

func TestStore_RunLifecycle(t *testing.T) {
	ctx := context.Background()
	hs := newTestStore(t)

	run, err := hs.StartRun(ctx, "run-001", "manual")
	require.NoError(t, err)
	assert.Equal(t, "run-001", run.ID)
	assert.Equal(t, "running", run.Status)
	assert.Nil(t, run.FinishedAt)

	require.NoError(t, hs.FinishRun(ctx, "run-001", "success"))

	got, err := hs.GetRun(ctx, "run-001")
	require.NoError(t, err)
	assert.Equal(t, "success", got.Status)
	assert.NotNil(t, got.FinishedAt)
}

func TestStore_AddAndGetResults(t *testing.T) {
	ctx := context.Background()
	hs := newTestStore(t)

	insertTestInstance(t, hs, "inst-1")
	insertTestInstance(t, hs, "inst-2")

	_, err := hs.StartRun(ctx, "run-002", "scheduler")
	require.NoError(t, err)

	diff := `{"before":{},"after":{}}`
	require.NoError(t, hs.AddResult(ctx, "run-002", "inst-1", "dns", "success", &diff, nil))
	errMsg := "connection refused"
	require.NoError(t, hs.AddResult(ctx, "run-002", "inst-2", "dhcp", "error", nil, &errMsg))

	results, err := hs.GetResults(ctx, "run-002")
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "success", results[0].Status)
	assert.Equal(t, "error", results[1].Status)
	assert.NotNil(t, results[1].ErrorMsg)
}

func TestStore_ListRuns(t *testing.T) {
	ctx := context.Background()
	hs := newTestStore(t)

	for _, id := range []string{"r1", "r2", "r3"} {
		_, err := hs.StartRun(ctx, id, "test")
		require.NoError(t, err)
	}

	runs, err := hs.ListRuns(ctx, 2, 0)
	require.NoError(t, err)
	assert.Len(t, runs, 2)

	all, err := hs.ListRuns(ctx, 10, 0)
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestStore_GetRun_NotFound(t *testing.T) {
	hs := newTestStore(t)
	_, err := hs.GetRun(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, history.ErrRunNotFound)
}
