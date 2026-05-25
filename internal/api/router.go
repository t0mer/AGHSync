package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/t0mer/aghsync/internal/api/handlers"
	"github.com/t0mer/aghsync/internal/api/middleware"
	"github.com/t0mer/aghsync/internal/config"
	"github.com/t0mer/aghsync/internal/instance"
	"github.com/t0mer/aghsync/internal/store"
)

// Deps holds the application dependencies threaded through the router.
type Deps struct {
	Store     *store.Store
	Config    *config.Config
	Logger    *slog.Logger
	Instances *instance.Repository
}

// NewRouter builds and returns the Chi router with all middleware and routes registered.
func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(middleware.RequestLogger(deps.Logger))
	r.Use(middleware.Recovery(deps.Logger))

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.APIAuth(deps.Config, deps.Logger))
		r.Use(middleware.CSRF(deps.Config))

		r.Get("/health", handlers.Health)

		// Instance management
		r.Get("/instances", handlers.ListInstances(deps.Instances))
		r.Post("/instances", handlers.CreateInstance(deps.Instances))
		r.Get("/instances/{id}", handlers.GetInstance(deps.Instances))
		r.Put("/instances/{id}", handlers.UpdateInstance(deps.Instances))
		r.Delete("/instances/{id}", handlers.DeleteInstance(deps.Instances))
		r.Put("/instances/{id}/promote", handlers.PromoteInstance(deps.Instances))
		r.Get("/instances/{id}/sync-config", handlers.GetSyncConfig(deps.Instances))
		r.Put("/instances/{id}/sync-config", handlers.UpdateSyncConfig(deps.Instances))

		// Settings
		r.Get("/settings", handlers.GetSettings(deps.Config))
		r.Put("/settings/ui-auth", handlers.UpdateUIAuth(deps.Config, deps.Logger))
		r.Post("/settings/api-token", handlers.GenerateAPIToken(deps.Config, deps.Logger))
		r.Delete("/settings/api-token", handlers.DeleteAPIToken(deps.Config, deps.Logger))
	})

	return r
}
