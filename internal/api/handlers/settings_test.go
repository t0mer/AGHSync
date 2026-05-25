package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/api/handlers"
	"github.com/t0mer/aghsync/internal/config"
	"github.com/t0mer/aghsync/internal/store"
)

func openConfig(t *testing.T) *config.Config {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return config.New(s)
}

// --- GetSettings ---

func TestGetSettings_Defaults(t *testing.T) {
	cfg := openConfig(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	w := httptest.NewRecorder()

	handlers.GetSettings(cfg)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, false, resp["ui_auth_enabled"])
	assert.Equal(t, false, resp["has_api_token"])
}

// --- UpdateUIAuth ---

func TestUpdateUIAuth_Enable(t *testing.T) {
	cfg := openConfig(t)
	body := `{"enabled":true,"username":"admin","password":"StrongPass1!"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/ui-auth", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.UpdateUIAuth(cfg, nil)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	enabled, _ := cfg.GetUIAuthEnabled()
	assert.True(t, enabled)
	username, _ := cfg.GetUIUsername()
	assert.Equal(t, "admin", username)
	hash, _ := cfg.GetUIPasswordHash()
	assert.NotEmpty(t, hash)
	// Hash should not be the plaintext
	assert.NotEqual(t, "StrongPass1!", hash)
}

func TestUpdateUIAuth_Disable(t *testing.T) {
	cfg := openConfig(t)
	require.NoError(t, cfg.SetUIAuthEnabled(true))

	body := `{"enabled":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/ui-auth", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	handlers.UpdateUIAuth(cfg, nil)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	enabled, _ := cfg.GetUIAuthEnabled()
	assert.False(t, enabled)
}

func TestUpdateUIAuth_EnableRequiresUsernameAndPassword(t *testing.T) {
	cfg := openConfig(t)
	body := `{"enabled":true,"username":"admin"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/ui-auth", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	handlers.UpdateUIAuth(cfg, nil)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- GenerateAPIToken ---

func TestGenerateAPIToken_ReturnsPlaintext(t *testing.T) {
	cfg := openConfig(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/api-token", nil)
	w := httptest.NewRecorder()

	handlers.GenerateAPIToken(cfg, nil)(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	token, ok := resp["token"].(string)
	require.True(t, ok)
	assert.Len(t, token, 64) // 32 bytes → 64 hex chars

	// Hash should be stored, not the plaintext
	hash, _ := cfg.GetAPITokenHash()
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, token, hash)
}

func TestGenerateAPIToken_EachCallProducesNewToken(t *testing.T) {
	cfg := openConfig(t)

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/settings/api-token", nil)
	w1 := httptest.NewRecorder()
	handlers.GenerateAPIToken(cfg, nil)(w1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/settings/api-token", nil)
	w2 := httptest.NewRecorder()
	handlers.GenerateAPIToken(cfg, nil)(w2, req2)

	var r1, r2 map[string]any
	json.NewDecoder(w1.Body).Decode(&r1)
	json.NewDecoder(w2.Body).Decode(&r2)
	assert.NotEqual(t, r1["token"], r2["token"])
}

// --- DeleteAPIToken ---

func TestDeleteAPIToken(t *testing.T) {
	cfg := openConfig(t)
	require.NoError(t, cfg.SetAPITokenHash("$2a$12$existing"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/settings/api-token", nil)
	w := httptest.NewRecorder()

	handlers.DeleteAPIToken(cfg, nil)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	hash, _ := cfg.GetAPITokenHash()
	assert.Empty(t, hash)
}
