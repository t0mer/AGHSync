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

func newSyncTestSetup(t *testing.T) (*internalsync.Dispatcher, *config.Config) {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	secret := make([]byte, 32)
	repo := instance.NewRepository(s.DB(), secret)
	hs := history.New(s.DB())
	engine := internalsync.NewEngine(repo, hs)
	d := internalsync.NewDispatcher(engine)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	_ = d.Start(ctx)

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
