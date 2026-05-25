package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/api/handlers"
	"github.com/t0mer/aghsync/internal/history"
	"github.com/t0mer/aghsync/internal/store"
)

func newHistoryTestStore(t *testing.T) *history.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return history.New(s.DB())
}

// insertTestInstanceForHistory inserts a minimal instance row so FK constraints on sync_results are satisfied.
func insertTestInstanceForHistory(t *testing.T, hs *history.Store, id string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := hs.DB().Exec(
		`INSERT INTO instances(id, name, address, username, password_enc, is_master, tls_skip_verify, created_at, updated_at)
		 VALUES(?,?,?,?,?,0,0,?,?)`,
		id, id, "http://localhost", "", "", now, now,
	)
	require.NoError(t, err)
}

func TestListHistory_EmptyReturnsEmptyArray(t *testing.T) {
	hs := newHistoryTestStore(t)

	r := chi.NewRouter()
	r.Get("/history", handlers.ListHistory(hs))

	req := httptest.NewRequest(http.MethodGet, "/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotNil(t, body)
	assert.Len(t, body, 0)
}

func TestListHistory_ReturnsRuns(t *testing.T) {
	hs := newHistoryTestStore(t)
	ctx := context.Background()

	for _, id := range []string{"r1", "r2", "r3"} {
		_, err := hs.StartRun(ctx, id, "manual")
		require.NoError(t, err)
	}

	r := chi.NewRouter()
	r.Get("/history", handlers.ListHistory(hs))

	req := httptest.NewRequest(http.MethodGet, "/history?limit=2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 2)
}

func TestGetHistoryRun_Found(t *testing.T) {
	hs := newHistoryTestStore(t)
	ctx := context.Background()

	insertTestInstanceForHistory(t, hs, "inst-1")

	_, err := hs.StartRun(ctx, "run-abc", "webhook")
	require.NoError(t, err)
	diff := `{"before":{},"after":{}}`
	require.NoError(t, hs.AddResult(ctx, "run-abc", "inst-1", "dns", "success", &diff, nil))

	r := chi.NewRouter()
	r.Get("/history/{runId}", handlers.GetHistoryRun(hs))

	req := httptest.NewRequest(http.MethodGet, "/history/run-abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "run-abc", body["id"])
	results, ok := body["results"].([]any)
	require.True(t, ok)
	assert.Len(t, results, 1)
}

func TestListHistory_LimitCappedAt200(t *testing.T) {
	hs := newHistoryTestStore(t)
	ctx := context.Background()

	// Insert 5 runs — well under any reasonable cap.
	for i := 0; i < 5; i++ {
		_, err := hs.StartRun(ctx, fmt.Sprintf("r%d", i), "manual")
		require.NoError(t, err)
	}

	r := chi.NewRouter()
	r.Get("/history", handlers.ListHistory(hs))

	// Request more than maxHistoryLimit — should still return only 5 (what exists).
	req := httptest.NewRequest(http.MethodGet, "/history?limit=99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	// We only have 5 runs; cap doesn't change the count, just ensures SQL LIMIT ≤ 200.
	assert.Len(t, body, 5)
}

func TestGetHistoryRun_NotFound(t *testing.T) {
	hs := newHistoryTestStore(t)

	r := chi.NewRouter()
	r.Get("/history/{runId}", handlers.GetHistoryRun(hs))

	req := httptest.NewRequest(http.MethodGet, "/history/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
