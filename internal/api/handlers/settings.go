package handlers

import (
	"log/slog"
	"net/http"

	"github.com/t0mer/aghsync/internal/auth"
	"github.com/t0mer/aghsync/internal/config"
)

type settingsResponse struct {
	UIAuthEnabled bool   `json:"ui_auth_enabled"`
	UIUsername    string `json:"ui_username"`
	HasAPIToken   bool   `json:"has_api_token"`
	SchedulerCron string `json:"scheduler_cron"`
	Port          int    `json:"port"`
	UITheme       string `json:"ui_theme"`
}

type updateThemeRequest struct {
	Theme string `json:"theme"`
}

type updateUIAuthRequest struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type generateTokenResponse struct {
	Token string `json:"token"`
}

// GetSettings returns current application settings (never exposes password hashes or tokens).
func GetSettings(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enabled, _ := cfg.GetUIAuthEnabled()
		username, _ := cfg.GetUIUsername()
		tokenHash, _ := cfg.GetAPITokenHash()
		cron, _ := cfg.GetSchedulerCron()
		port, _ := cfg.GetPort()
		theme, _ := cfg.GetUITheme()
		WriteJSON(w, http.StatusOK, settingsResponse{
			UIAuthEnabled: enabled,
			UIUsername:    username,
			HasAPIToken:   tokenHash != "",
			SchedulerCron: cron,
			Port:          port,
			UITheme:       theme,
		})
	}
}

// UpdateTheme persists the user's light/dark preference.
func UpdateTheme(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updateThemeRequest
		if err := DecodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Theme != "dark" && req.Theme != "light" {
			WriteError(w, http.StatusBadRequest, "theme must be \"dark\" or \"light\"")
			return
		}
		if err := cfg.SetUITheme(req.Theme); err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to save theme")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// UpdateUIAuth enables or disables Basic Auth for the UI.
// When enabling, username and password are required.
func UpdateUIAuth(cfg *config.Config, _ *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updateUIAuthRequest
		if err := DecodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Enabled {
			if req.Username == "" || req.Password == "" {
				WriteError(w, http.StatusBadRequest, "username and password are required when enabling UI auth")
				return
			}
			hash, err := auth.HashPassword(req.Password)
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "failed to hash password")
				return
			}
			if err := cfg.SetUIUsername(req.Username); err != nil {
				WriteError(w, http.StatusInternalServerError, "failed to save username")
				return
			}
			if err := cfg.SetUIPasswordHash(hash); err != nil {
				WriteError(w, http.StatusInternalServerError, "failed to save password hash")
				return
			}
		}
		if err := cfg.SetUIAuthEnabled(req.Enabled); err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to update ui auth setting")
			return
		}
		GetSettings(cfg)(w, r)
	}
}

// GenerateAPIToken creates a new API token. The plaintext is returned once; only
// the bcrypt hash is stored. Any previous token is replaced.
func GenerateAPIToken(cfg *config.Config, _ *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plain, hash, err := auth.GenerateToken()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to generate token")
			return
		}
		if err := cfg.SetAPITokenHash(hash); err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to store token hash")
			return
		}
		WriteJSON(w, http.StatusCreated, generateTokenResponse{Token: plain})
	}
}

// DeleteAPIToken removes the API token, returning the system to bootstrap mode.
func DeleteAPIToken(cfg *config.Config, _ *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := cfg.SetAPITokenHash(""); err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to delete token")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
