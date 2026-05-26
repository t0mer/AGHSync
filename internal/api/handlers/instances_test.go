package handlers_test

import (
	"bytes"
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
	"github.com/t0mer/aghsync/internal/instance"
	"github.com/t0mer/aghsync/internal/store"
)

func openInstanceRepo(t *testing.T) *instance.Repository {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return instance.NewRepository(s.DB(), make([]byte, 32))
}

func chiCtxWithID(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- ListInstances ---

func TestListInstances_Empty(t *testing.T) {
	repo := openInstanceRepo(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	w := httptest.NewRecorder()

	handlers.ListInstances(repo)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Empty(t, body)
}

func TestListInstances_ReturnsAll(t *testing.T) {
	repo := openInstanceRepo(t)
	ctx := context.Background()
	repo.Create(ctx, "A", "http://1.1.1.1:3000", "u", "p", true, false)
	repo.Create(ctx, "B", "http://2.2.2.2:3000", "u", "p", false, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	w := httptest.NewRecorder()

	handlers.ListInstances(repo)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 2)
	// Password must not appear in responses
	for _, inst := range body {
		_, hasPassword := inst["password"]
		assert.False(t, hasPassword)
		_, hasPasswordEnc := inst["password_enc"]
		assert.False(t, hasPasswordEnc)
	}
}

// --- CreateInstance ---

func TestCreateInstance_Valid(t *testing.T) {
	repo := openInstanceRepo(t)
	body := `{"name":"Master","address":"http://10.0.0.1:3000","username":"admin","password":"secret","is_master":true,"tls_skip_verify":false}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.CreateInstance(repo)(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["id"])
	assert.Equal(t, "Master", resp["name"])
	assert.True(t, resp["is_master"].(bool))
}

func TestCreateInstance_MissingName(t *testing.T) {
	repo := openInstanceRepo(t)
	body := `{"address":"http://10.0.0.1:3000","password":"p"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	handlers.CreateInstance(repo)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateInstance_MissingAddress(t *testing.T) {
	repo := openInstanceRepo(t)
	body := `{"name":"X","password":"p"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	handlers.CreateInstance(repo)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- GetInstance ---

func TestGetInstance_NotFound(t *testing.T) {
	repo := openInstanceRepo(t)
	req := chiCtxWithID(
		httptest.NewRequest(http.MethodGet, "/api/v1/instances/bad-id", nil),
		"bad-id",
	)
	w := httptest.NewRecorder()
	handlers.GetInstance(repo)(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetInstance_Found(t *testing.T) {
	repo := openInstanceRepo(t)
	inst, _ := repo.Create(context.Background(), "X", "http://1.1.1.1:3000", "u", "p", false, false)

	req := chiCtxWithID(
		httptest.NewRequest(http.MethodGet, "/api/v1/instances/"+inst.ID, nil),
		inst.ID,
	)
	w := httptest.NewRecorder()
	handlers.GetInstance(repo)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, inst.ID, resp["id"])
}

// --- UpdateInstance ---

func TestUpdateInstance_Valid(t *testing.T) {
	repo := openInstanceRepo(t)
	inst, _ := repo.Create(context.Background(), "Old", "http://1.1.1.1:3000", "u", "p", false, false)

	body := `{"name":"New","address":"http://2.2.2.2:3000","username":"u2","tls_skip_verify":true}`
	req := chiCtxWithID(
		httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(body)),
		inst.ID,
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.UpdateInstance(repo)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "New", resp["name"])
}

// --- DeleteInstance ---

func TestDeleteInstance_Valid(t *testing.T) {
	repo := openInstanceRepo(t)
	inst, _ := repo.Create(context.Background(), "Del", "http://1.1.1.1:3000", "u", "p", false, false)

	req := chiCtxWithID(
		httptest.NewRequest(http.MethodDelete, "/", nil),
		inst.ID,
	)
	w := httptest.NewRecorder()
	handlers.DeleteInstance(repo)(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteInstance_NotFound(t *testing.T) {
	repo := openInstanceRepo(t)
	req := chiCtxWithID(
		httptest.NewRequest(http.MethodDelete, "/", nil),
		"nope",
	)
	w := httptest.NewRecorder()
	handlers.DeleteInstance(repo)(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- PromoteInstance ---

func TestPromoteInstance_Valid(t *testing.T) {
	repo := openInstanceRepo(t)
	ctx := context.Background()
	repo.Create(ctx, "Master", "http://1.1.1.1:3000", "u", "p", true, false)
	child, _ := repo.Create(ctx, "Child", "http://2.2.2.2:3000", "u", "p", false, false)

	req := chiCtxWithID(
		httptest.NewRequest(http.MethodPut, "/", nil),
		child.ID,
	)
	w := httptest.NewRecorder()
	handlers.PromoteInstance(repo)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- SyncConfig ---

func TestGetSyncConfig_Valid(t *testing.T) {
	repo := openInstanceRepo(t)
	inst, _ := repo.Create(context.Background(), "C", "http://1.1.1.1:3000", "u", "p", false, false)

	req := chiCtxWithID(
		httptest.NewRequest(http.MethodGet, "/", nil),
		inst.ID,
	)
	w := httptest.NewRecorder()
	handlers.GetSyncConfig(repo)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp, len(instance.AllConfigTypes))
}

func TestUpdateSyncConfig_Valid(t *testing.T) {
	repo := openInstanceRepo(t)
	inst, _ := repo.Create(context.Background(), "C", "http://1.1.1.1:3000", "u", "p", false, false)

	body := `{"config":[{"config_type":"filtering","enabled":false},{"config_type":"dns","enabled":true}]}`
	req := chiCtxWithID(
		httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(body)),
		inst.ID,
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.UpdateSyncConfig(repo)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// Ensure timestamps are RFC3339 in responses (not Go default format).
func TestCreateInstance_TimestampsRFC3339(t *testing.T) {
	repo := openInstanceRepo(t)
	body := `{"name":"T","address":"http://1.1.1.1:3000","username":"u","password":"p"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handlers.CreateInstance(repo)(w, req)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	// Should parse as RFC3339 without error
	_, err := time.Parse(time.RFC3339, resp["created_at"].(string))
	assert.NoError(t, err)
}

// --- TestConnectionHandler ---

func startFakeAGH(t *testing.T, loginStatus int) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/control/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(loginStatus)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestTestConnectionHandler_Success(t *testing.T) {
	aghURL := startFakeAGH(t, http.StatusOK)
	body := fmt.Sprintf(`{"address":%q,"username":"admin","password":"secret","tls_skip_verify":false}`, aghURL)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/test-connection", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.TestConnectionHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, true, resp["ok"])
}

func TestTestConnectionHandler_BadCredentials(t *testing.T) {
	aghURL := startFakeAGH(t, http.StatusUnauthorized)
	body := fmt.Sprintf(`{"address":%q,"username":"admin","password":"wrong","tls_skip_verify":false}`, aghURL)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/test-connection", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.TestConnectionHandler(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "invalid username or password")
}

func TestTestConnectionHandler_MissingAddress(t *testing.T) {
	body := `{"username":"admin","password":"secret","tls_skip_verify":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/test-connection", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.TestConnectionHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTestConnectionHandler_InvalidScheme(t *testing.T) {
	body := `{"address":"file:///etc/passwd","username":"u","password":"p","tls_skip_verify":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/test-connection", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.TestConnectionHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "http or https")
}

func TestTestConnectionHandler_Unreachable(t *testing.T) {
	body := `{"address":"http://127.0.0.1:1","username":"u","password":"p","tls_skip_verify":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/test-connection", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.TestConnectionHandler(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["error"])
}
