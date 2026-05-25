package config

import (
	"crypto/rand"
	"database/sql"
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

// Get returns the value for key, or "" if not set.
func (c *Config) Get(key string) (string, error) {
	var val string
	err := c.s.DB().QueryRow("SELECT value FROM app_config WHERE key=?", key).Scan(&val)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return val, err
}

// GetWithDefault returns the stored value, or def if the key is absent.
func (c *Config) GetWithDefault(key, def string) (string, error) {
	val, err := c.Get(key)
	if err != nil || val != "" {
		return val, err
	}
	return def, nil
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

// InstallSecret returns the per-install 32-byte random secret used for AES-GCM key derivation.
// Generated once on first call and persisted.
func (c *Config) InstallSecret() ([]byte, error) {
	val, err := c.Get("install_secret")
	if err != nil {
		return nil, err
	}
	if val != "" {
		return []byte(val), nil
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate install secret: %w", err)
	}
	if err := c.Set("install_secret", string(secret)); err != nil {
		return nil, err
	}
	return secret, nil
}
