package history

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

const timeFmt = time.RFC3339

var ErrRunNotFound = errors.New("sync run not found")

type Run struct {
	ID          string     `json:"id"`
	TriggeredBy string     `json:"triggered_by"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Status      string     `json:"status"`
}

type Result struct {
	ID         string    `json:"id"`
	RunID      string    `json:"run_id"`
	InstanceID string    `json:"instance_id"`
	ConfigType string    `json:"config_type"`
	Status     string    `json:"status"`
	DiffJSON   *string   `json:"diff_json,omitempty"`
	ErrorMsg   *string   `json:"error_msg,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Store struct {
	db *sql.DB
}

// New creates a Store backed by the given database handle.
func New(db *sql.DB) *Store { return &Store{db: db} }

// DB returns the raw *sql.DB (used in tests to insert fixture rows).
func (s *Store) DB() *sql.DB { return s.db }

// StartRun inserts a new sync run with status "running" and returns it.
func (s *Store) StartRun(ctx context.Context, runID, triggeredBy string) (*Run, error) {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sync_runs(id, triggered_by, started_at, status) VALUES(?,?,?,?)`,
		runID, triggeredBy, now.Format(timeFmt), "running",
	)
	if err != nil {
		return nil, err
	}
	return &Run{ID: runID, TriggeredBy: triggeredBy, StartedAt: now, Status: "running"}, nil
}

// FinishRun updates the run status and records finished_at.
func (s *Store) FinishRun(ctx context.Context, runID, status string) error {
	now := time.Now().UTC().Format(timeFmt)
	res, err := s.db.ExecContext(ctx,
		`UPDATE sync_runs SET status=?, finished_at=? WHERE id=?`, status, now, runID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRunNotFound
	}
	return nil
}

// AddResult records the outcome of syncing one config type to one instance within a run.
// diffJSON and errorMsg are optional.
func (s *Store) AddResult(ctx context.Context, runID, instanceID, configType, status string, diffJSON, errorMsg *string) error {
	id := uuid.NewString()
	now := time.Now().UTC().Format(timeFmt)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sync_results(id, run_id, instance_id, config_type, status, diff_json, error_msg, created_at)
		 VALUES(?,?,?,?,?,?,?,?)`,
		id, runID, instanceID, configType, status, diffJSON, errorMsg, now,
	)
	return err
}

// ListRuns returns sync runs ordered by started_at DESC, with pagination.
func (s *Store) ListRuns(ctx context.Context, limit, offset int) ([]*Run, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, triggered_by, started_at, finished_at, status
		 FROM sync_runs ORDER BY started_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []*Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// GetRun fetches a single sync run by ID. Returns ErrRunNotFound if absent.
func (s *Store) GetRun(ctx context.Context, runID string) (*Run, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, triggered_by, started_at, finished_at, status FROM sync_runs WHERE id=?`, runID)
	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRunNotFound
	}
	return r, err
}

// GetResults fetches all results for a run in ascending created_at order.
func (s *Store) GetResults(ctx context.Context, runID string) ([]*Result, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, run_id, instance_id, config_type, status, diff_json, error_msg, created_at
		 FROM sync_results WHERE run_id=? ORDER BY created_at ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*Result
	for rows.Next() {
		var res Result
		var diff, errMsg sql.NullString
		var createdAt string
		if err := rows.Scan(&res.ID, &res.RunID, &res.InstanceID, &res.ConfigType,
			&res.Status, &diff, &errMsg, &createdAt); err != nil {
			return nil, err
		}
		if diff.Valid {
			res.DiffJSON = &diff.String
		}
		if errMsg.Valid {
			res.ErrorMsg = &errMsg.String
		}
		res.CreatedAt, _ = time.Parse(timeFmt, createdAt)
		results = append(results, &res)
	}
	return results, rows.Err()
}

// --- internal helpers ---

type scanner interface{ Scan(dest ...any) error }

func scanRun(s scanner) (*Run, error) {
	var r Run
	var finishedAt sql.NullString
	var startedAt string
	if err := s.Scan(&r.ID, &r.TriggeredBy, &startedAt, &finishedAt, &r.Status); err != nil {
		return nil, err
	}
	r.StartedAt, _ = time.Parse(timeFmt, startedAt)
	if finishedAt.Valid {
		t, _ := time.Parse(timeFmt, finishedAt.String)
		r.FinishedAt = &t
	}
	return &r, nil
}
