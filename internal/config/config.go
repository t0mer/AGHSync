package config

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/t0mer/aghsync/internal/store"
)

const (
	defaultPort     = 8080
	defaultLogLevel = "warning"
)

// Config provides typed access to the app_config table.
type Config struct {
	s *store.Store
}

// New returns a Config backed by the given store.
func New(s *store.Store) *Config { return &Config{s: s} }

// Get returns the value for key and whether it was found.
func (c *Config) Get(key string) (string, bool, error) {
	var val string
	err := c.s.DB().QueryRow("SELECT value FROM app_config WHERE key=?", key).Scan(&val)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

// GetWithDefault returns the stored value, or def if the key is absent.
func (c *Config) GetWithDefault(key, def string) (string, error) {
	val, found, err := c.Get(key)
	if err != nil {
		return "", err
	}
	if !found {
		return def, nil
	}
	return val, nil
}

// Set upserts a key-value pair into app_config.
func (c *Config) Set(key, value string) error {
	_, err := c.s.DB().Exec(
		`INSERT INTO app_config(key,value) VALUES(?,?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value,
	)
	return err
}

// GetPort returns the configured port, defaulting to 8080.
func (c *Config) GetPort() (int, error) {
	val, err := c.GetWithDefault("port", strconv.Itoa(defaultPort))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val)
}

// SetPort persists the port.
func (c *Config) SetPort(port int) error { return c.Set("port", strconv.Itoa(port)) }

// GetLogLevel returns the configured log level, defaulting to "warning".
func (c *Config) GetLogLevel() (string, error) {
	return c.GetWithDefault("log_level", defaultLogLevel)
}

// SetLogLevel persists the log level.
func (c *Config) SetLogLevel(level string) error { return c.Set("log_level", level) }

// GetSchedulerCron returns the configured cron expression, or "" if not set.
func (c *Config) GetSchedulerCron() (string, error) {
	return c.GetWithDefault("scheduler_cron", "")
}

// SetSchedulerCron persists the cron expression.
func (c *Config) SetSchedulerCron(expr string) error { return c.Set("scheduler_cron", expr) }

// GetUITheme returns the preferred UI theme ("dark", "light", or "" when not set).
func (c *Config) GetUITheme() (string, error) {
	return c.GetWithDefault("ui_theme", "")
}

// SetUITheme persists the UI theme preference.
func (c *Config) SetUITheme(theme string) error { return c.Set("ui_theme", theme) }

// GetUIAuthEnabled returns whether UI basic auth is enabled (default false).
func (c *Config) GetUIAuthEnabled() (bool, error) {
	val, err := c.GetWithDefault("ui_auth_enabled", "0")
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

// SetUIAuthEnabled persists the UI auth enabled flag.
func (c *Config) SetUIAuthEnabled(enabled bool) error {
	v := "0"
	if enabled {
		v = "1"
	}
	return c.Set("ui_auth_enabled", v)
}

// GetUIUsername returns the configured UI username (default "").
func (c *Config) GetUIUsername() (string, error) {
	return c.GetWithDefault("ui_username", "")
}

// SetUIUsername persists the UI username.
func (c *Config) SetUIUsername(u string) error { return c.Set("ui_username", u) }

// GetUIPasswordHash returns the bcrypt hash of the UI password (default "").
func (c *Config) GetUIPasswordHash() (string, error) {
	return c.GetWithDefault("ui_password_hash", "")
}

// SetUIPasswordHash persists the bcrypt password hash.
func (c *Config) SetUIPasswordHash(h string) error { return c.Set("ui_password_hash", h) }

// GetAPITokenHash returns the bcrypt hash of the API token (default "").
// Empty string means no token is configured (bootstrap mode).
func (c *Config) GetAPITokenHash() (string, error) {
	return c.GetWithDefault("api_token_hash", "")
}

// SetAPITokenHash persists the API token hash. Pass "" to remove (return to bootstrap mode).
func (c *Config) SetAPITokenHash(h string) error {
	if h == "" {
		_, err := c.s.DB().Exec(`DELETE FROM app_config WHERE key='api_token_hash'`)
		return err
	}
	return c.Set("api_token_hash", h)
}

// InstallSecret returns the per-install 32-byte random secret used for AES-GCM key derivation.
// Generated once on first call and persisted as hex.
func (c *Config) InstallSecret() ([]byte, error) {
	val, found, err := c.Get("install_secret")
	if err != nil {
		return nil, err
	}
	if found {
		return hex.DecodeString(val)
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate install secret: %w", err)
	}
	encoded := hex.EncodeToString(secret)
	if err := c.Set("install_secret", encoded); err != nil {
		return nil, err
	}
	return secret, nil
}
