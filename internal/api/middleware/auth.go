package middleware

import (
	"log/slog"
	"net/http"

	"github.com/t0mer/aghsync/internal/auth"
	"github.com/t0mer/aghsync/internal/config"
)

// APIAuth protects /api/v1/* routes.
// Bootstrap mode (no token hash stored): all requests pass through.
// Once a token is configured: requests must provide X-API-Token, OR
// valid Basic auth credentials when UI auth is enabled.
func APIAuth(cfg *config.Config, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenHash, err := cfg.GetAPITokenHash()
			if err != nil {
				if logger != nil {
					logger.Error("APIAuth: read token hash", "err", err)
				}
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			// Bootstrap mode: no token configured yet
			if tokenHash == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check X-API-Token header first
			if token := r.Header.Get("X-API-Token"); token != "" {
				if auth.CheckToken(token, tokenHash) {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Fall back to Basic auth when UI auth is enabled
			uiEnabled, err := cfg.GetUIAuthEnabled()
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if uiEnabled {
				u, p, ok := r.BasicAuth()
				if ok {
					username, _ := cfg.GetUIUsername()
					pwHash, _ := cfg.GetUIPasswordHash()
					if u == username && auth.CheckPassword(p, pwHash) {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			w.Header().Set("WWW-Authenticate", `Basic realm="aghsync"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		})
	}
}

// UIAuth protects non-API routes with Basic auth when ui_auth_enabled is true.
func UIAuth(cfg *config.Config, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enabled, err := cfg.GetUIAuthEnabled()
			if err != nil {
				if logger != nil {
					logger.Error("UIAuth: read ui_auth_enabled", "err", err)
				}
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if !enabled {
				next.ServeHTTP(w, r)
				return
			}

			u, p, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="aghsync"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			username, _ := cfg.GetUIUsername()
			pwHash, _ := cfg.GetUIPasswordHash()
			if u != username || !auth.CheckPassword(p, pwHash) {
				w.Header().Set("WWW-Authenticate", `Basic realm="aghsync"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
