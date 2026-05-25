package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/api/handlers"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	handlers.WriteJSON(w, http.StatusCreated, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusCreated, w.Result().StatusCode)
	assert.Equal(t, "application/json", w.Result().Header.Get("Content-Type"))
	assert.JSONEq(t, `{"key":"value"}`, w.Body.String())
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	handlers.WriteError(w, http.StatusBadRequest, "something went wrong")

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	assert.JSONEq(t, `{"error":"something went wrong"}`, w.Body.String())
}

func TestDecodeJSON_Valid(t *testing.T) {
	body := `{"name":"test","value":42}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var dst struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	require.NoError(t, handlers.DecodeJSON(r, &dst))
	assert.Equal(t, "test", dst.Name)
	assert.Equal(t, 42, dst.Value)
}

func TestDecodeJSON_UnknownField(t *testing.T) {
	body := `{"name":"test","unexpected_field":"x"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	var dst struct {
		Name string `json:"name"`
	}
	err := handlers.DecodeJSON(r, &dst)
	assert.Error(t, err)
}
