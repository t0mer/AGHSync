package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/t0mer/aghsync/internal/api/middleware"
	"github.com/t0mer/aghsync/internal/config"
	"github.com/t0mer/aghsync/internal/store"
)

func newCSRFConfig(t *testing.T) *config.Config {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return config.New(s)
}

func TestCSRF_GetRequest_AlwaysPasses(t *testing.T) {
	cfg := newCSRFConfig(t)
	cfg.SetUIAuthEnabled(true) // CSRF only matters when UI auth is enabled

	mw := middleware.CSRF(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	// No X-Requested-With header
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRF_PostWithHeader_Passes(t *testing.T) {
	cfg := newCSRFConfig(t)
	cfg.SetUIAuthEnabled(true)

	mw := middleware.CSRF(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRF_PostWithoutHeader_WhenUIAuthEnabled_Forbidden(t *testing.T) {
	cfg := newCSRFConfig(t)
	cfg.SetUIAuthEnabled(true)

	mw := middleware.CSRF(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", nil)
	// No X-Requested-With
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRF_PostWithoutHeader_WhenUIAuthDisabled_Passes(t *testing.T) {
	cfg := newCSRFConfig(t)
	// ui_auth_enabled defaults to false

	mw := middleware.CSRF(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRF_DeleteWithoutHeader_WhenUIAuthEnabled_Forbidden(t *testing.T) {
	cfg := newCSRFConfig(t)
	cfg.SetUIAuthEnabled(true)

	mw := middleware.CSRF(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/instances/123", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRF_PutWithHeader_Passes(t *testing.T) {
	cfg := newCSRFConfig(t)
	cfg.SetUIAuthEnabled(true)

	mw := middleware.CSRF(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodPut, "/api/v1/instances/123", nil)
	req.Header.Set("X-Requested-With", "fetch")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
