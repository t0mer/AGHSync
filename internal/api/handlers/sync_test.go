package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/api/handlers"
	"github.com/t0mer/aghsync/internal/config"
	"github.com/t0mer/aghsync/internal/history"
	"github.com/t0mer/aghsync/internal/instance"
	internalsync "github.com/t0mer/aghsync/internal/sync"
	"github.com/t0mer/aghsync/internal/store"
)

// minimalAGHSrv returns a test server that satisfies the AGH API contract.
func minimalAGHSrv(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
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

func newSyncTestSetup(t *testing.T) (*internalsync.Dispatcher, *config.Config) {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	secret := make([]byte, 32)
	repo := instance.NewRepository(s.DB(), secret)
	hs := history.New(s.DB())

	// Seed master + slave so the pre-flight HasEnabledSlaves check passes.
	masterSrv := minimalAGHSrv(t)
	slaveSrv := minimalAGHSrv(t)
	ctx := context.Background()
	_, err = repo.Create(ctx, "master", masterSrv.URL, "u", "p", true, false)
	require.NoError(t, err)
	_, err = repo.Create(ctx, "slave", slaveSrv.URL, "u", "p", false, false)
	require.NoError(t, err)

	engine := internalsync.NewEngine(repo, hs)
	d := internalsync.NewDispatcher(engine)
	cancelCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	_ = d.Start(cancelCtx)

	cfg := config.New(s)
	return d, cfg
}

func TestTriggerSync_Returns202WithRunID(t *testing.T) {
	d, _ := newSyncTestSetup(t)

	r := chi.NewRouter()
	r.Post("/sync/run", handlers.TriggerSync(d))

	req := httptest.NewRequest(http.MethodPost, "/sync/run", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotEmpty(t, body["run_id"])
}

func TestGetSyncStatus_ReturnsJSON(t *testing.T) {
	d, _ := newSyncTestSetup(t)

	r := chi.NewRouter()
	r.Get("/sync/status", handlers.GetSyncStatus(d))

	req := httptest.NewRequest(http.MethodGet, "/sync/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Contains(t, body, "current")
	assert.Contains(t, body, "last")
}

func TestSetSchedule_ValidExpression(t *testing.T) {
	d, cfg := newSyncTestSetup(t)
	sched := internalsync.NewScheduler(d)
	sched.Start()
	defer sched.Stop()

	r := chi.NewRouter()
	r.Put("/sync/schedule", handlers.SetSchedule(cfg, sched))

	body := bytes.NewBufferString(`{"cron":"0 * * * *"}`)
	req := httptest.NewRequest(http.MethodPut, "/sync/schedule", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "0 * * * *", resp["cron"])
}

func TestSetSchedule_InvalidExpression_Returns400(t *testing.T) {
	d, cfg := newSyncTestSetup(t)
	sched := internalsync.NewScheduler(d)

	r := chi.NewRouter()
	r.Put("/sync/schedule", handlers.SetSchedule(cfg, sched))

	body := bytes.NewBufferString(`{"cron":"not-valid"}`)
	req := httptest.NewRequest(http.MethodPut, "/sync/schedule", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetSchedule_Empty_Disables(t *testing.T) {
	d, cfg := newSyncTestSetup(t)
	sched := internalsync.NewScheduler(d)

	r := chi.NewRouter()
	r.Put("/sync/schedule", handlers.SetSchedule(cfg, sched))

	body := bytes.NewBufferString(`{"cron":""}`)
	req := httptest.NewRequest(http.MethodPut, "/sync/schedule", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTriggerSync_Returns409WhenBusy(t *testing.T) {
	// Build a dispatcher without starting it so the queue stays full.
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	secret := make([]byte, 32)
	repo := instance.NewRepository(s.DB(), secret)
	hs := history.New(s.DB())

	// Seed master + slave so the pre-flight passes and Submit can queue.
	masterSrv := minimalAGHSrv(t)
	slaveSrv := minimalAGHSrv(t)
	ctx := context.Background()
	_, err = repo.Create(ctx, "master", masterSrv.URL, "u", "p", true, false)
	require.NoError(t, err)
	_, err = repo.Create(ctx, "slave", slaveSrv.URL, "u", "p", false, false)
	require.NoError(t, err)

	engine := internalsync.NewEngine(repo, hs)
	d := internalsync.NewDispatcher(engine)
	// No d.Start() — queue is drained by nobody.

	r := chi.NewRouter()
	r.Post("/sync/run", handlers.TriggerSync(d))

	// First request fills the queue.
	req1 := httptest.NewRequest(http.MethodPost, "/sync/run", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusAccepted, w1.Code)

	// Second request must see ErrSyncBusy → 409.
	req2 := httptest.NewRequest(http.MethodPost, "/sync/run", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusConflict, w2.Code)
}
