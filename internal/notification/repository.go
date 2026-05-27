package notification

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/t0mer/aghsync/internal/auth"
)

const timeFmt = time.RFC3339

// Repository provides CRUD operations over the notification_channels table.
type Repository struct {
	db            *sql.DB
	installSecret []byte
}

// NewRepository creates a Repository. installSecret is used for AES-256-GCM config encryption.
func NewRepository(db *sql.DB, installSecret []byte) *Repository {
	return &Repository{db: db, installSecret: installSecret}
}

func isDuplicateNameErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: notification_channels.name")
}

// Create inserts a new notification channel. config is stored AES-256-GCM encrypted.
func (r *Repository) Create(ctx context.Context, name string, chType ChannelType, config string, notifySuccess, notifyFailure, enabled bool) (*Channel, error) {
	enc, err := auth.EncryptPassword(config, r.installSecret)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	now := time.Now().UTC()
	nowFmt := now.Format(timeFmt)
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO notification_channels(id,name,type,config_enc,notify_success,notify_failure,enabled,created_at,updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		id, name, string(chType), enc, btoi(notifySuccess), btoi(notifyFailure), btoi(enabled), nowFmt, nowFmt,
	)
	if err != nil {
		if isDuplicateNameErr(err) {
			return nil, ErrDuplicateName
		}
		return nil, err
	}
	return &Channel{
		ID: id, Name: name, Type: chType, Config: config,
		NotifySuccess: notifySuccess, NotifyFailure: notifyFailure, Enabled: enabled,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// Get returns a single channel by ID with decrypted config.
func (r *Repository) Get(ctx context.Context, id string) (*Channel, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id,name,type,config_enc,notify_success,notify_failure,enabled,created_at,updated_at
		 FROM notification_channels WHERE id=?`, id)
	ch, err := r.scanChannel(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return ch, err
}

// List returns all channels ordered by created_at ascending, configs decrypted.
func (r *Repository) List(ctx context.Context) ([]*Channel, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id,name,type,config_enc,notify_success,notify_failure,enabled,created_at,updated_at
		 FROM notification_channels ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Channel
	for rows.Next() {
		ch, err := r.scanChannel(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, ch)
	}
	return list, rows.Err()
}

// Update modifies an existing channel. config may be "" to keep the existing encrypted value.
func (r *Repository) Update(ctx context.Context, id, name string, chType ChannelType, config string, notifySuccess, notifyFailure, enabled bool) (*Channel, error) {
	now := time.Now().UTC().Format(timeFmt)
	var err error
	if config != "" {
		var enc string
		enc, err = auth.EncryptPassword(config, r.installSecret)
		if err != nil {
			return nil, err
		}
		_, err = r.db.ExecContext(ctx,
			`UPDATE notification_channels SET name=?,type=?,config_enc=?,notify_success=?,notify_failure=?,enabled=?,updated_at=? WHERE id=?`,
			name, string(chType), enc, btoi(notifySuccess), btoi(notifyFailure), btoi(enabled), now, id)
	} else {
		_, err = r.db.ExecContext(ctx,
			`UPDATE notification_channels SET name=?,type=?,notify_success=?,notify_failure=?,enabled=?,updated_at=? WHERE id=?`,
			name, string(chType), btoi(notifySuccess), btoi(notifyFailure), btoi(enabled), now, id)
	}
	if err != nil {
		if isDuplicateNameErr(err) {
			return nil, ErrDuplicateName
		}
		return nil, err
	}
	return r.Get(ctx, id)
}

// Delete removes a channel by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM notification_channels WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListEnabled returns all enabled channels, decrypted, filtered by the trigger type.
// onSuccess=true → channels that have notify_success=1; similarly for onSuccess=false.
func (r *Repository) ListEnabled(ctx context.Context, onSuccess bool) ([]*Channel, error) {
	col := "notify_failure"
	if onSuccess {
		col = "notify_success"
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id,name,type,config_enc,notify_success,notify_failure,enabled,created_at,updated_at
		 FROM notification_channels WHERE enabled=1 AND `+col+`=1 ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Channel
	for rows.Next() {
		ch, err := r.scanChannel(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, ch)
	}
	return list, rows.Err()
}

// --- internal helpers ---

type scanner interface{ Scan(dest ...any) error }

func (r *Repository) scanChannel(s scanner) (*Channel, error) {
	var ch Channel
	var chType string
	var notifySuccess, notifyFailure, enabled int
	var configEnc, createdAt, updatedAt string
	if err := s.Scan(&ch.ID, &ch.Name, &chType, &configEnc,
		&notifySuccess, &notifyFailure, &enabled, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	plain, err := auth.DecryptPassword(configEnc, r.installSecret)
	if err != nil {
		return nil, err
	}
	ch.Type = ChannelType(chType)
	ch.Config = plain
	ch.NotifySuccess = notifySuccess == 1
	ch.NotifyFailure = notifyFailure == 1
	ch.Enabled = enabled == 1
	ch.CreatedAt, _ = time.Parse(timeFmt, createdAt)
	ch.UpdatedAt, _ = time.Parse(timeFmt, updatedAt)
	return &ch, nil
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
