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
	Version    int            `json:"version"`
	ExportedAt string         `json:"exported_at"`
	Config     BackupConfig   `json:"config"`
	Instances  []BackupInstance `json:"instances"`
}

// BackupConfig holds auth and scheduler settings.
type BackupConfig struct {
	UIAuthEnabled  bool   `json:"ui_auth_enabled"`
	UIUsername     string `json:"ui_username"`
	UIPasswordHash string `json:"ui_password_hash"`
	APITokenHash   string `json:"api_token_hash"`
	SchedulerCron  string `json:"scheduler_cron"`
	UITheme        string `json:"ui_theme"`
	// InstallSecret is the hex-encoded AES key seed used to encrypt instance
	// passwords. Must be restored alongside the encrypted passwords so they
	// remain decryptable on any machine.
	InstallSecret  string `json:"install_secret"`
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
	// Include the raw hex secret so encrypted instance passwords can be
	// decrypted after restoring to a different machine.
	installSecret, _, _ := cfg.Get("install_secret")

	data.Config = BackupConfig{
		UIAuthEnabled:  authEnabled,
		UIUsername:     uiUsername,
		UIPasswordHash: uiPasswordHash,
		APITokenHash:   apiTokenHash,
		SchedulerCron:  schedulerCron,
		UITheme:        uiTheme,
		InstallSecret:  installSecret,
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
