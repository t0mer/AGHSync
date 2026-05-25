package middleware

import (
	"net/http"

	"github.com/t0mer/aghsync/internal/config"
)

var mutatingMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// CSRF blocks mutating requests that lack an X-Requested-With header when UI
// auth is enabled. This prevents cross-site form submissions using the
// browser's cached Basic auth credentials.
func CSRF(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mutatingMethods[r.Method] {
				next.ServeHTTP(w, r)
				return
			}
			enabled, err := cfg.GetUIAuthEnabled()
			if err != nil || !enabled {
				next.ServeHTTP(w, r)
				return
			}
			if r.Header.Get("X-Requested-With") == "" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
