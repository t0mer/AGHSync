package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/t0mer/aghsync/internal/api/docs"
	"github.com/t0mer/aghsync/internal/api/handlers"
	"github.com/t0mer/aghsync/internal/api/middleware"
	"github.com/t0mer/aghsync/internal/config"
	"github.com/t0mer/aghsync/internal/history"
	"github.com/t0mer/aghsync/internal/instance"
	internalsync "github.com/t0mer/aghsync/internal/sync"
	"github.com/t0mer/aghsync/internal/store"
)

// Deps holds the application dependencies threaded through the router.
type Deps struct {
	Store      *store.Store
	Config     *config.Config
	Logger     *slog.Logger
	Instances  *instance.Repository
	History    *history.Store
	Dispatcher *internalsync.Dispatcher
	Scheduler  *internalsync.Scheduler
}

// NewRouter builds and returns the Chi router with all middleware and routes registered.
func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(middleware.RequestLogger(deps.Logger))
	r.Use(middleware.Recovery(deps.Logger))

	// Health check — unauthenticated (liveness probe).
	r.Get("/api/v1/health", handlers.Health)

	// API docs — no auth required, served outside /api/v1 group.
	docsHandler := docs.SwaggerUIHandler()
	r.Get("/api/docs/openapi.yaml", docs.Spec)
	r.Get("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/docs/", http.StatusFound)
	})
	r.Handle("/api/docs/*", docsHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.APIAuth(deps.Config, deps.Logger))
		r.Use(middleware.CSRF(deps.Config))

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

		// Sync
		r.Post("/sync/run", handlers.TriggerSync(deps.Dispatcher))
		r.Get("/sync/status", handlers.GetSyncStatus(deps.Dispatcher))
		r.Put("/sync/schedule", handlers.SetSchedule(deps.Config, deps.Scheduler))

		// Webhook trigger (same auth as /api/v1 — token or basic auth)
		r.Post("/webhook/sync", handlers.TriggerWebhookSync(deps.Dispatcher))

		// History
		r.Get("/history", handlers.ListHistory(deps.History))
		r.Get("/history/{runId}", handlers.GetHistoryRun(deps.History))
	})

	return r
}
