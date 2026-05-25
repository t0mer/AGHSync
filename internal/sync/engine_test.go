package sync_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/history"
	"github.com/t0mer/aghsync/internal/instance"
	internalsync "github.com/t0mer/aghsync/internal/sync"
	"github.com/t0mer/aghsync/internal/store"
)

// minimalAGHServer returns a test server that returns {} for any GET and 200 for any POST/PUT.
func minimalAGHServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			// rewrite snapshot needs a list endpoint too
			if r.URL.Path == "/control/rewrite/list" {
				w.Write([]byte(`[]`))
				return
			}
			w.Write([]byte(`{"enabled":false}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestSetup(t *testing.T) (*instance.Repository, *history.Store, []byte) {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	installSecret := make([]byte, 32)
	repo := instance.NewRepository(s.DB(), installSecret)
	hs := history.New(s.DB())
	return repo, hs, installSecret
}

func TestEngine_Run_NoMaster(t *testing.T) {
	repo, hs, _ := newTestSetup(t)
	engine := internalsync.NewEngine(repo, hs)

	runID := uuid.NewString()
	err := engine.Run(context.Background(), runID, "test")
	assert.ErrorIs(t, err, internalsync.ErrNoMaster, "should fail with ErrNoMaster when no master configured")
}

func TestEngine_Run_MasterNoChildren(t *testing.T) {
	repo, hs, _ := newTestSetup(t)
	srv := minimalAGHServer(t)

	_, err := repo.Create(context.Background(), "master", srv.URL, "admin", "pass", true, false)
	require.NoError(t, err)

	engine := internalsync.NewEngine(repo, hs)
	runID := uuid.NewString()
	err = engine.Run(context.Background(), runID, "manual")
	require.NoError(t, err, "run with no children should succeed")

	run, err := hs.GetRun(context.Background(), runID)
	require.NoError(t, err)
	assert.Equal(t, "success", run.Status)
}

func TestEngine_Run_SyncsChildInstances(t *testing.T) {
	repo, hs, _ := newTestSetup(t)
	masterSrv := minimalAGHServer(t)
	childSrv := minimalAGHServer(t)

	_, err := repo.Create(context.Background(), "master", masterSrv.URL, "u", "p", true, false)
	require.NoError(t, err)
	childInst, err := repo.Create(context.Background(), "child", childSrv.URL, "u", "p", false, false)
	require.NoError(t, err)

	engine := internalsync.NewEngine(repo, hs)
	runID := uuid.NewString()
	err = engine.Run(context.Background(), runID, "manual")
	require.NoError(t, err)

	results, err := hs.GetResults(context.Background(), runID)
	require.NoError(t, err)
	assert.NotEmpty(t, results, "should have results for child instance")
	for _, r := range results {
		assert.Equal(t, childInst.ID, r.InstanceID)
	}
}

func TestEngine_Run_RecordsErrorForUnreachableChild(t *testing.T) {
	repo, hs, _ := newTestSetup(t)
	masterSrv := minimalAGHServer(t)

	_, err := repo.Create(context.Background(), "master", masterSrv.URL, "u", "p", true, false)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), "bad-child", "http://127.0.0.1:1", "u", "p", false, false)
	require.NoError(t, err)

	engine := internalsync.NewEngine(repo, hs)
	runID := uuid.NewString()
	_ = engine.Run(context.Background(), runID, "manual")

	run, err := hs.GetRun(context.Background(), runID)
	require.NoError(t, err)
	assert.NotEqual(t, "running", run.Status)

	results, err := hs.GetResults(context.Background(), runID)
	require.NoError(t, err)
	hasError := false
	for _, r := range results {
		if r.Status == "error" {
			hasError = true
		}
	}
	assert.True(t, hasError, "should record at least one error result for unreachable child")
}
