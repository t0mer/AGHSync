package middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/t0mer/aghsync/internal/api/middleware"
	"github.com/t0mer/aghsync/internal/auth"
	"github.com/t0mer/aghsync/internal/config"
	"github.com/t0mer/aghsync/internal/store"
)

func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return config.New(s)
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// --- APIAuth ---

func TestAPIAuth_Bootstrap_NoTokenConfigured(t *testing.T) {
	// No token hash stored → bootstrap mode → all requests pass through
	cfg := newTestConfig(t)
	mw := middleware.APIAuth(cfg, slog.Default())(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIAuth_ValidToken(t *testing.T) {
	cfg := newTestConfig(t)
	plain, hash, _ := auth.GenerateToken()
	cfg.SetAPITokenHash(hash)

	mw := middleware.APIAuth(cfg, slog.Default())(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	req.Header.Set("X-API-Token", plain)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIAuth_InvalidToken(t *testing.T) {
	cfg := newTestConfig(t)
	_, hash, _ := auth.GenerateToken()
	cfg.SetAPITokenHash(hash)

	mw := middleware.APIAuth(cfg, slog.Default())(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	req.Header.Set("X-API-Token", "bad-token")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIAuth_NoTokenHeader_Unauthorized(t *testing.T) {
	cfg := newTestConfig(t)
	_, hash, _ := auth.GenerateToken()
	cfg.SetAPITokenHash(hash)

	mw := middleware.APIAuth(cfg, slog.Default())(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIAuth_BasicAuthAccepted_WhenUIAuthEnabled(t *testing.T) {
	cfg := newTestConfig(t)
	_, hash, _ := auth.GenerateToken()
	cfg.SetAPITokenHash(hash) // token is set, so not bootstrap

	pwHash, _ := auth.HashPassword("password123")
	cfg.SetUIAuthEnabled(true)
	cfg.SetUIUsername("admin")
	cfg.SetUIPasswordHash(pwHash)

	mw := middleware.APIAuth(cfg, slog.Default())(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	req.SetBasicAuth("admin", "password123")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIAuth_BasicAuthRejected_WrongPassword(t *testing.T) {
	cfg := newTestConfig(t)
	_, hash, _ := auth.GenerateToken()
	cfg.SetAPITokenHash(hash)

	pwHash, _ := auth.HashPassword("password123")
	cfg.SetUIAuthEnabled(true)
	cfg.SetUIUsername("admin")
	cfg.SetUIPasswordHash(pwHash)

	mw := middleware.APIAuth(cfg, slog.Default())(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	req.SetBasicAuth("admin", "wrongpassword")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- UIAuth ---

func TestUIAuth_Disabled_PassesThrough(t *testing.T) {
	cfg := newTestConfig(t)
	// ui_auth_enabled defaults to false
	mw := middleware.UIAuth(cfg, slog.Default())(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUIAuth_Enabled_NoCredentials_Unauthorized(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.SetUIAuthEnabled(true)
	cfg.SetUIUsername("admin")
	hash, _ := auth.HashPassword("pass")
	cfg.SetUIPasswordHash(hash)

	mw := middleware.UIAuth(cfg, slog.Default())(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUIAuth_Enabled_ValidCredentials(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.SetUIAuthEnabled(true)
	cfg.SetUIUsername("admin")
	hash, _ := auth.HashPassword("pass")
	cfg.SetUIPasswordHash(hash)

	mw := middleware.UIAuth(cfg, slog.Default())(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "pass")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUIAuth_Enabled_WrongPassword_Unauthorized(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.SetUIAuthEnabled(true)
	cfg.SetUIUsername("admin")
	hash, _ := auth.HashPassword("pass")
	cfg.SetUIPasswordHash(hash)

	mw := middleware.UIAuth(cfg, slog.Default())(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "wrong")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
