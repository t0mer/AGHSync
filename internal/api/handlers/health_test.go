package handlers_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/api"
	"github.com/t0mer/aghsync/internal/api/handlers"
	"github.com/t0mer/aghsync/internal/config"
	"github.com/t0mer/aghsync/internal/history"
	"github.com/t0mer/aghsync/internal/instance"
	internalsync "github.com/t0mer/aghsync/internal/sync"
	"github.com/t0mer/aghsync/internal/store"
)

func newTestDeps(t *testing.T) api.Deps {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	cfg := config.New(s)
	secret, err := cfg.InstallSecret()
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := instance.NewRepository(s.DB(), secret)
	hs := history.New(s.DB())
	engine := internalsync.NewEngine(repo, hs)
	dispatcher := internalsync.NewDispatcher(engine)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	_ = dispatcher.Start(ctx)
	scheduler := internalsync.NewScheduler(dispatcher)

	return api.Deps{
		Store:      s,
		Config:     cfg,
		Logger:     logger,
		Instances:  repo,
		History:    hs,
		Dispatcher: dispatcher,
		Scheduler:  scheduler,
	}
}

func TestHealth_ReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handlers.Health(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

func TestRouter_Health_MethodNotAllowed(t *testing.T) {
	router := api.NewRouter(newTestDeps(t))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Result().StatusCode)
}

func TestRouter_InstanceRoutes_Exist(t *testing.T) {
	router := api.NewRouter(newTestDeps(t))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
	assert.NotEqual(t, http.StatusMethodNotAllowed, w.Code)
}

func TestRouter_SettingsRoute_Exists(t *testing.T) {
	router := api.NewRouter(newTestDeps(t))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
	assert.NotEqual(t, http.StatusMethodNotAllowed, w.Code)
}

func TestRouter_DocsRoutes_NoAuth(t *testing.T) {
	router := api.NewRouter(newTestDeps(t))

	tests := []struct {
		path string
		want int
	}{
		{"/api/docs/", http.StatusOK},
		{"/api/docs/openapi.yaml", http.StatusOK},
		{"/api/docs/swagger-ui.css", http.StatusOK},
		{"/api/docs/swagger-ui-bundle.js", http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Errorf("GET %s: expected %d, got %d", tc.path, tc.want, w.Code)
			}
		})
	}
}

func TestRouter_DocsRedirect(t *testing.T) {
	router := api.NewRouter(newTestDeps(t))

	req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expected 301, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/api/docs/" {
		t.Errorf("expected Location /api/docs/, got %q", loc)
	}
}

func TestRouter_SyncAndHistoryRoutes_Exist(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)

	// Insert a fixture run so GET /history/{runId} returns 200 rather than
	// the handler's own 404-for-not-found, which would fail the route-exists check.
	fixtureRunID := "fixture-run-id-for-route-test"
	_, err := deps.Store.DB().Exec(
		`INSERT INTO sync_runs(id, triggered_by, started_at, status) VALUES(?,?,datetime('now'),?)`,
		fixtureRunID, "test", "success",
	)
	require.NoError(t, err)

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/sync/run"},
		{http.MethodGet, "/api/v1/sync/status"},
		{http.MethodPut, "/api/v1/sync/schedule"},
		{http.MethodGet, "/api/v1/history"},
		{http.MethodGet, "/api/v1/history/" + fixtureRunID},
	} {
		req := httptest.NewRequest(route.method, route.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusNotFound, w.Code, "route %s %s should be registered", route.method, route.path)
	}
}
