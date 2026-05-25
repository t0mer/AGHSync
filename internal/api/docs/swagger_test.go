package docs_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/t0mer/aghsync/internal/api/docs"
)

func TestOpenAPISpec_NotEmpty(t *testing.T) {
	if len(docs.OpenAPISpec) == 0 {
		t.Fatal("OpenAPISpec is empty — missing internal/api/docs/openapi.yaml?")
	}
	content := string(docs.OpenAPISpec)
	if !strings.HasPrefix(content, "openapi:") {
		t.Fatalf("expected spec to start with 'openapi:', got: %q", content[:min(len(content), 30)])
	}
}

func TestSpec_ServesYAML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/docs/openapi.yaml", nil)
	w := httptest.NewRecorder()
	docs.Spec(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/yaml" {
		t.Fatalf("expected Content-Type application/yaml, got %q", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.HasPrefix(string(body), "openapi:") {
		t.Fatalf("response body does not start with 'openapi:'")
	}
}

func TestSwaggerUIHandler_ServesIndex(t *testing.T) {
	handler := docs.SwaggerUIHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/docs/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected Content-Type text/html, got %q", ct)
	}
}

func TestSwaggerUIHandler_ServesCSS(t *testing.T) {
	handler := docs.SwaggerUIHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/docs/swagger-ui.css", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for swagger-ui.css, got %d", resp.StatusCode)
	}
}

func TestSwaggerUIHandler_ServesJS(t *testing.T) {
	handler := docs.SwaggerUIHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/docs/swagger-ui-bundle.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for swagger-ui-bundle.js, got %d", resp.StatusCode)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
