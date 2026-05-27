package backup

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/t0mer/aghsync/internal/config"
)

const schemaVersion = 1

// BackupData is the serialisable representation of a full AGHSync backup.
type BackupData struct {
	Version       int                         `json:"version"`
	ExportedAt    string                      `json:"exported_at"`
	Config        BackupConfig                `json:"config"`
	Instances     []BackupInstance            `json:"instances"`
	Notifications []BackupNotificationChannel `json:"notifications"`
}

// BackupNotificationChannel holds one notification_channels row.
type BackupNotificationChannel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	ConfigEnc     string `json:"config_enc"`
	NotifySuccess bool   `json:"notify_success"`
	NotifyFailure bool   `json:"notify_failure"`
	Enabled       bool   `json:"enabled"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// BackupConfig holds auth and scheduler settings.
type BackupConfig struct {
	UIAuthEnabled   bool   `json:"ui_auth_enabled"`
	UIUsername      string `json:"ui_username"`
	UIPasswordHash  string `json:"ui_password_hash"`
	APITokenHash    string `json:"api_token_hash"`
	SchedulerCron   string `json:"scheduler_cron"`
	UITheme         string `json:"ui_theme"`
	WatchdogEnabled bool   `json:"watchdog_enabled"`
	WatchdogPath    string `json:"watchdog_path"`
	// InstallSecret is the hex-encoded AES key seed used to encrypt instance
	// passwords. Must be restored alongside the encrypted passwords so they
	// remain decryptable on any machine.
	InstallSecret string `json:"install_secret"`
}

// BackupInstance holds one instance plus its sync config.
type BackupInstance struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Address       string              `json:"address"`
	Username      string              `json:"username"`
	PasswordEnc   string              `json:"password_enc"`
	IsMaster      bool                `json:"is_master"`
	TLSSkipVerify bool                `json:"tls_skip_verify"`
	CreatedAt     string              `json:"created_at"`
	UpdatedAt     string              `json:"updated_at"`
	SyncConfig    []BackupSyncConfig  `json:"sync_config"`
}

// BackupSyncConfig is one sync-config row.
type BackupSyncConfig struct {
	ConfigType string `json:"config_type"`
	Enabled    bool   `json:"enabled"`
}

// Export builds a BackupData snapshot of the current application state.
func Export(ctx context.Context, db *sql.DB, cfg *config.Config) (*BackupData, error) {
	data := &BackupData{
		Version:    schemaVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// --- app config ---
	authEnabled, _ := cfg.GetUIAuthEnabled()
	uiUsername, _ := cfg.GetUIUsername()
	uiPasswordHash, _ := cfg.GetUIPasswordHash()
	apiTokenHash, _ := cfg.GetAPITokenHash()
	schedulerCron, _ := cfg.GetSchedulerCron()
	uiTheme, _ := cfg.GetUITheme()
	watchdogEnabled, _ := cfg.GetWatchdogEnabled()
	watchdogPath, _ := cfg.GetWatchdogPath()
	// Include the raw hex secret so encrypted instance passwords can be
	// decrypted after restoring to a different machine.
	installSecret, _, _ := cfg.Get("install_secret")

	data.Config = BackupConfig{
		UIAuthEnabled:   authEnabled,
		UIUsername:      uiUsername,
		UIPasswordHash:  uiPasswordHash,
		APITokenHash:    apiTokenHash,
		SchedulerCron:   schedulerCron,
		UITheme:         uiTheme,
		WatchdogEnabled: watchdogEnabled,
		WatchdogPath:    watchdogPath,
		InstallSecret:   installSecret,
	}

	// --- instances (load all before opening sync_config queries to avoid SQLite deadlock) ---
	rows, err := db.QueryContext(ctx,
		`SELECT id, name, address, username, password_enc, is_master, tls_skip_verify, created_at, updated_at
		 FROM instances ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("query instances: %w", err)
	}
	for rows.Next() {
		var inst BackupInstance
		var isMaster, tlsSkip int
		if err := rows.Scan(
			&inst.ID, &inst.Name, &inst.Address, &inst.Username, &inst.PasswordEnc,
			&isMaster, &tlsSkip, &inst.CreatedAt, &inst.UpdatedAt,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan instance: %w", err)
		}
		inst.IsMaster = isMaster == 1
		inst.TLSSkipVerify = tlsSkip == 1
		data.Instances = append(data.Instances, inst)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// --- sync_config (separate pass; outer rows are closed above) ---
	for i := range data.Instances {
		scRows, err := db.QueryContext(ctx,
			`SELECT config_type, enabled FROM sync_config WHERE instance_id=? ORDER BY config_type ASC`,
			data.Instances[i].ID)
		if err != nil {
			return nil, fmt.Errorf("query sync_config for %s: %w", data.Instances[i].ID, err)
		}
		for scRows.Next() {
			var sc BackupSyncConfig
			var enabled int
			if err := scRows.Scan(&sc.ConfigType, &enabled); err != nil {
				scRows.Close()
				return nil, fmt.Errorf("scan sync_config: %w", err)
			}
			sc.Enabled = enabled == 1
			data.Instances[i].SyncConfig = append(data.Instances[i].SyncConfig, sc)
		}
		scRows.Close()
		if err := scRows.Err(); err != nil {
			return nil, err
		}
	}

	// --- notification channels ---
	ncRows, err := db.QueryContext(ctx,
		`SELECT id, name, type, config_enc, notify_success, notify_failure, enabled, created_at, updated_at
		 FROM notification_channels ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("query notification_channels: %w", err)
	}
	defer ncRows.Close()
	for ncRows.Next() {
		var nc BackupNotificationChannel
		var notifySuccess, notifyFailure, enabled int
		if err := ncRows.Scan(
			&nc.ID, &nc.Name, &nc.Type, &nc.ConfigEnc,
			&notifySuccess, &notifyFailure, &enabled,
			&nc.CreatedAt, &nc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification_channel: %w", err)
		}
		nc.NotifySuccess = notifySuccess == 1
		nc.NotifyFailure = notifyFailure == 1
		nc.Enabled = enabled == 1
		data.Notifications = append(data.Notifications, nc)
	}
	if err := ncRows.Err(); err != nil {
		return nil, err
	}

	return data, nil
}

// Restore replaces instances and configuration with the contents of data.
// Existing instances are removed (which cascades to their sync_results);
// sync_runs history rows are preserved.
func Restore(ctx context.Context, db *sql.DB, cfg *config.Config, data *BackupData) error {
	if data.Version != schemaVersion {
		return fmt.Errorf("unsupported backup version %d", data.Version)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove existing data.
	if _, err = tx.ExecContext(ctx, `DELETE FROM notification_channels`); err != nil {
		return fmt.Errorf("clear notification_channels: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM sync_config`); err != nil {
		return fmt.Errorf("clear sync_config: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM instances`); err != nil {
		return fmt.Errorf("clear instances: %w", err)
	}

	// Restore instances and their sync configs.
	for _, inst := range data.Instances {
		if _, err = tx.ExecContext(ctx,
			`INSERT INTO instances(id, name, address, username, password_enc, is_master, tls_skip_verify, created_at, updated_at)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			inst.ID, inst.Name, inst.Address, inst.Username, inst.PasswordEnc,
			btoi(inst.IsMaster), btoi(inst.TLSSkipVerify), inst.CreatedAt, inst.UpdatedAt,
		); err != nil {
			return fmt.Errorf("restore instance %q: %w", inst.Name, err)
		}
		for _, sc := range inst.SyncConfig {
			if _, err = tx.ExecContext(ctx,
				`INSERT INTO sync_config(instance_id, config_type, enabled) VALUES(?,?,?)`,
				inst.ID, sc.ConfigType, btoi(sc.Enabled),
			); err != nil {
				return fmt.Errorf("restore sync_config for %q/%s: %w", inst.Name, sc.ConfigType, err)
			}
		}
	}

	// Restore notification channels.
	for _, nc := range data.Notifications {
		if _, err = tx.ExecContext(ctx,
			`INSERT INTO notification_channels(id, name, type, config_enc, notify_success, notify_failure, enabled, created_at, updated_at)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			nc.ID, nc.Name, nc.Type, nc.ConfigEnc,
			btoi(nc.NotifySuccess), btoi(nc.NotifyFailure), btoi(nc.Enabled),
			nc.CreatedAt, nc.UpdatedAt,
		); err != nil {
			return fmt.Errorf("restore notification_channel %q: %w", nc.Name, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	// Restore app config (individual upserts; safe outside the instance transaction).
	if err = cfg.SetUIAuthEnabled(data.Config.UIAuthEnabled); err != nil {
		return fmt.Errorf("restore ui_auth_enabled: %w", err)
	}
	if err = cfg.SetUIUsername(data.Config.UIUsername); err != nil {
		return fmt.Errorf("restore ui_username: %w", err)
	}
	if err = cfg.SetUIPasswordHash(data.Config.UIPasswordHash); err != nil {
		return fmt.Errorf("restore ui_password_hash: %w", err)
	}
	if err = cfg.SetAPITokenHash(data.Config.APITokenHash); err != nil {
		return fmt.Errorf("restore api_token_hash: %w", err)
	}
	if err = cfg.SetSchedulerCron(data.Config.SchedulerCron); err != nil {
		return fmt.Errorf("restore scheduler_cron: %w", err)
	}
	if data.Config.UITheme != "" {
		if err = cfg.SetUITheme(data.Config.UITheme); err != nil {
			return fmt.Errorf("restore ui_theme: %w", err)
		}
	}
	if err = cfg.SetWatchdogEnabled(data.Config.WatchdogEnabled); err != nil {
		return fmt.Errorf("restore watchdog_enabled: %w", err)
	}
	if err = cfg.SetWatchdogPath(data.Config.WatchdogPath); err != nil {
		return fmt.Errorf("restore watchdog_path: %w", err)
	}
	// Restore the install secret so encrypted instance passwords remain
	// decryptable on the destination machine.
	if data.Config.InstallSecret != "" {
		if err = cfg.Set("install_secret", data.Config.InstallSecret); err != nil {
			return fmt.Errorf("restore install_secret: %w", err)
		}
	}

	return nil
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
