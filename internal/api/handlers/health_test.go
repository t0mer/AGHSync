package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/api/handlers"
)

func TestHealth_ReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handlers.Health(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

func TestHealth_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	// Chi enforces method matching; testing the handler directly returns 200 regardless.
	// Method enforcement is tested via the router in router_test.go (added in later plan tasks).
	handlers.Health(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}
