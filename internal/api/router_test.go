package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRouter_SPACatchAll(t *testing.T) {
	fakeFS := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<html>app</html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('hi')")},
	}

	handler := spaHandler(fakeFS)

	tests := []struct {
		path         string
		wantContains string
		wantCode     int
	}{
		{"/", "<html>app</html>", http.StatusOK},
		{"/instances", "<html>app</html>", http.StatusOK},
		{"/history/abc-123", "<html>app</html>", http.StatusOK},
		{"/assets/app.js", "console.log('hi')", http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tc.wantCode {
				t.Errorf("path %s: got status %d, want %d", tc.path, w.Code, tc.wantCode)
			}
			body := w.Body.String()
			if !strings.Contains(body, tc.wantContains) {
				t.Errorf("path %s: body %q does not contain %q", tc.path, body, tc.wantContains)
			}
		})
	}
}
