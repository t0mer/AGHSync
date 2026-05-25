package instance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/t0mer/aghsync/internal/auth"
)

// ErrNotFound is returned when an instance does not exist.
var ErrNotFound = errors.New("instance not found")

const timeFmt = "2006-01-02 15:04:05"

// Repository provides CRUD and sync-config operations over the instances table.
type Repository struct {
	db            *sql.DB
	installSecret []byte
}

// NewRepository creates a Repository. installSecret is used for AES-256-GCM password encryption.
func NewRepository(db *sql.DB, installSecret []byte) *Repository {
	return &Repository{db: db, installSecret: installSecret}
}

// Create inserts a new instance. If isMaster is true, the current master (if any) is demoted.
// All config types are enabled by default for non-master instances.
func (r *Repository) Create(ctx context.Context, name, address, username, password string, isMaster, tlsSkipVerify bool) (*Instance, error) {
	enc, err := auth.EncryptPassword(password, r.installSecret)
	if err != nil {
		return nil, fmt.Errorf("encrypt password: %w", err)
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	nowFmt := now.Format(timeFmt)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if isMaster {
		if _, err = tx.ExecContext(ctx,
			`UPDATE instances SET is_master=0, updated_at=? WHERE is_master=1`, nowFmt); err != nil {
			return nil, err
		}
	}

	if _, err = tx.ExecContext(ctx,
		`INSERT INTO instances(id, name, address, username, password_enc, is_master, tls_skip_verify, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		id, name, address, username, enc, btoi(isMaster), btoi(tlsSkipVerify), nowFmt, nowFmt,
	); err != nil {
		return nil, err
	}

	if !isMaster {
		for _, ct := range AllConfigTypes {
			if _, err = tx.ExecContext(ctx,
				`INSERT INTO sync_config(instance_id, config_type, enabled) VALUES(?,?,1)`,
				id, ct,
			); err != nil {
				return nil, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &Instance{
		ID: id, Name: name, Address: address, Username: username,
		IsMaster: isMaster, TLSSkipVerify: tlsSkipVerify,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// Get returns a single instance by ID.
func (r *Repository) Get(ctx context.Context, id string) (*Instance, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, address, username, is_master, tls_skip_verify, created_at, updated_at
		 FROM instances WHERE id=?`, id)
	inst, err := scanInstance(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return inst, err
}

// List returns all instances ordered by created_at ascending.
func (r *Repository) List(ctx context.Context) ([]*Instance, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, address, username, is_master, tls_skip_verify, created_at, updated_at
		 FROM instances ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Instance
	for rows.Next() {
		inst, err := scanInstance(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, inst)
	}
	return list, rows.Err()
}

// Update modifies name, address, username, tls_skip_verify, and optionally password.
// Pass nil for password to keep the existing encrypted value.
func (r *Repository) Update(ctx context.Context, id, name, address, username string, password *string, tlsSkipVerify bool) (*Instance, error) {
	now := time.Now().UTC().Format(timeFmt)

	var (
		res sql.Result
		err error
	)
	if password != nil {
		enc, encErr := auth.EncryptPassword(*password, r.installSecret)
		if encErr != nil {
			return nil, fmt.Errorf("encrypt password: %w", encErr)
		}
		res, err = r.db.ExecContext(ctx,
			`UPDATE instances SET name=?, address=?, username=?, password_enc=?, tls_skip_verify=?, updated_at=? WHERE id=?`,
			name, address, username, enc, btoi(tlsSkipVerify), now, id)
	} else {
		res, err = r.db.ExecContext(ctx,
			`UPDATE instances SET name=?, address=?, username=?, tls_skip_verify=?, updated_at=? WHERE id=?`,
			name, address, username, btoi(tlsSkipVerify), now, id)
	}
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}

	return r.Get(ctx, id)
}

// Delete removes an instance by ID (cascade deletes sync_config + sync_results rows).
func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM instances WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Promote atomically demotes the current master and promotes the given instance.
func (r *Repository) Promote(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(timeFmt)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM instances WHERE id=?`, id).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return ErrNotFound
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE instances SET is_master=0, updated_at=? WHERE is_master=1`, now); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx,
		`UPDATE instances SET is_master=1, updated_at=? WHERE id=?`, now, id); err != nil {
		return err
	}
	return tx.Commit()
}

// GetSyncConfig returns the enabled config types for a child instance.
func (r *Repository) GetSyncConfig(ctx context.Context, id string) ([]SyncConfigEntry, error) {
	if _, err := r.Get(ctx, id); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT config_type, enabled FROM sync_config WHERE instance_id=? ORDER BY config_type ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []SyncConfigEntry
	for rows.Next() {
		var e SyncConfigEntry
		var enabled int
		if err := rows.Scan(&e.ConfigType, &enabled); err != nil {
			return nil, err
		}
		e.Enabled = enabled == 1
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// SetSyncConfig upserts the given config entries for an instance.
func (r *Repository) SetSyncConfig(ctx context.Context, id string, entries []SyncConfigEntry) error {
	if _, err := r.Get(ctx, id); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, e := range entries {
		if _, err = tx.ExecContext(ctx,
			`INSERT INTO sync_config(instance_id, config_type, enabled) VALUES(?,?,?)
			 ON CONFLICT(instance_id, config_type) DO UPDATE SET enabled=excluded.enabled`,
			id, e.ConfigType, btoi(e.Enabled),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// --- internal helpers ---

type scanner interface{ Scan(dest ...any) error }

func scanInstance(s scanner) (*Instance, error) {
	var inst Instance
	var isMaster, tlsSkipVerify int
	var createdAt, updatedAt string
	if err := s.Scan(
		&inst.ID, &inst.Name, &inst.Address, &inst.Username,
		&isMaster, &tlsSkipVerify, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	inst.IsMaster = isMaster == 1
	inst.TLSSkipVerify = tlsSkipVerify == 1
	inst.CreatedAt, _ = time.Parse(timeFmt, createdAt)
	inst.UpdatedAt, _ = time.Parse(timeFmt, updatedAt)
	return &inst, nil
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
