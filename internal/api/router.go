package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

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
	"github.com/t0mer/aghsync/internal/webui"
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
	Watchdog   *internalsync.Watchdog
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
		r.Post("/instances/test-connection", handlers.TestConnectionHandler)
		r.Get("/instances/statuses", handlers.GetInstancesStatuses(deps.Instances))
		r.Get("/instances/{id}", handlers.GetInstance(deps.Instances))
		r.Put("/instances/{id}", handlers.UpdateInstance(deps.Instances))
		r.Delete("/instances/{id}", handlers.DeleteInstance(deps.Instances))
		r.Put("/instances/{id}/promote", handlers.PromoteInstance(deps.Instances))
		r.Get("/instances/{id}/stats", handlers.GetInstanceStats(deps.Instances))
		r.Get("/instances/{id}/sync-config", handlers.GetSyncConfig(deps.Instances))
		r.Put("/instances/{id}/sync-config", handlers.UpdateSyncConfig(deps.Instances))

		// Settings
		r.Get("/settings", handlers.GetSettings(deps.Config))
		r.Put("/settings/ui-auth", handlers.UpdateUIAuth(deps.Config, deps.Logger))
		r.Post("/settings/api-token", handlers.GenerateAPIToken(deps.Config, deps.Logger))
		r.Delete("/settings/api-token", handlers.DeleteAPIToken(deps.Config, deps.Logger))
		r.Put("/settings/theme", handlers.UpdateTheme(deps.Config))
		r.Put("/settings/watchdog", handlers.UpdateWatchdog(deps.Config, deps.Watchdog))

		// Sync
		r.Post("/sync/run", handlers.TriggerSync(deps.Dispatcher))
		r.Get("/sync/status", handlers.GetSyncStatus(deps.Dispatcher))
		r.Put("/sync/schedule", handlers.SetSchedule(deps.Config, deps.Scheduler))

		// Webhook trigger (same auth as /api/v1 — token or basic auth)
		r.Post("/webhook/sync", handlers.TriggerWebhookSync(deps.Dispatcher))

		// History
		r.Get("/history", handlers.ListHistory(deps.History))
		r.Get("/history/{runId}", handlers.GetHistoryRun(deps.History))

		// Backup / Restore
		r.Get("/backup/export", handlers.ExportBackup(deps.Store.DB(), deps.Config))
		r.Post("/backup/restore", handlers.RestoreBackup(deps.Store.DB(), deps.Config))
	})

	// SPA catch-all: serve static assets; fall back to index.html for client-side routes.
	sub, err := fs.Sub(webui.FS, "dist")
	if err != nil {
		panic("webui: cannot sub dist: " + err.Error())
	}
	r.Handle("/*", spaHandler(sub))

	return r
}

// spaHandler serves files from the embedded FS. For paths that do not exist
// in the FS (i.e. client-side React Router routes), it falls back to index.html
// so the browser-side router can take over.
func spaHandler(fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(fsys, path); err != nil {
			http.ServeFileFS(w, r, fsys, "index.html")
			return
		}
		http.FileServerFS(fsys).ServeHTTP(w, r)
	})
}
