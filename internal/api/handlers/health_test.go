package handlers_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/api"
	"github.com/t0mer/aghsync/internal/api/handlers"
	"github.com/t0mer/aghsync/internal/config"
	"github.com/t0mer/aghsync/internal/instance"
	"github.com/t0mer/aghsync/internal/store"
)

func newTestDeps(t *testing.T) api.Deps {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	cfg := config.New(s)
	secret, _ := cfg.InstallSecret()
	return api.Deps{
		Store:     s,
		Config:    cfg,
		Logger:    slog.Default(),
		Instances: instance.NewRepository(s.DB(), secret),
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
