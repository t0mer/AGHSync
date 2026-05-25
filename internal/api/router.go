package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/t0mer/aghsync/internal/api/handlers"
	"github.com/t0mer/aghsync/internal/api/middleware"
)

// NewRouter builds and returns the Chi router with all middleware and routes registered.
func NewRouter(logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.Recovery(logger))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", handlers.Health)
	})

	return r
}
