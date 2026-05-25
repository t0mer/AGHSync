package docs

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed openapi.yaml
var OpenAPISpec []byte

//go:embed swagger-ui
var swaggerUIFS embed.FS

// SwaggerUIHandler returns an http.Handler that serves the offline Swagger UI
// from embedded assets. Paths are expected to be full (e.g. /api/docs/swagger-ui.css);
// the handler strips the /api/docs prefix internally.
func SwaggerUIHandler() http.Handler {
	sub, _ := fs.Sub(swaggerUIFS, "swagger-ui")
	return http.StripPrefix("/api/docs", http.FileServer(http.FS(sub)))
}

// Spec serves the raw OpenAPI YAML spec with Content-Type: application/yaml.
func Spec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(OpenAPISpec)
}
