package sync

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrSyncBusy = errors.New("a sync operation is already in progress")

// RunStatus describes the state of one sync run.
type RunStatus struct {
	RunID       string     `json:"run_id"`
	TriggeredBy string     `json:"triggered_by"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Status      string     `json:"status"`
}

type dispatchReq struct {
	runID       string
	triggeredBy string
}

// Dispatcher serializes sync requests so only one sync runs at a time.
type Dispatcher struct {
	engine  *Engine
	queue   chan dispatchReq // buffered(1): one pending slot
	mu      sync.RWMutex
	current *RunStatus
	last    *RunStatus
}

// NewDispatcher creates a Dispatcher. Call Start to begin processing.
func NewDispatcher(engine *Engine) *Dispatcher {
	return &Dispatcher{
		engine: engine,
		queue:  make(chan dispatchReq, 1),
	}
}

// Start begins the dispatch loop in a goroutine. Cancel ctx to stop.
// The returned channel is closed when the loop goroutine exits, allowing
// callers to wait for any in-flight run to finish before process exit.
func (d *Dispatcher) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.loop(ctx)
	}()
	return done
}

func (d *Dispatcher) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-d.queue:
			now := time.Now().UTC()
			d.mu.Lock()
			d.current = &RunStatus{
				RunID:       req.runID,
				TriggeredBy: req.triggeredBy,
				StartedAt:   now,
				Status:      "running",
			}
			d.mu.Unlock()

			err := d.engine.Run(ctx, req.runID, req.triggeredBy)

			fin := time.Now().UTC()
			d.mu.Lock()
			// ErrNoSlaves means nothing ran and nothing was written to history —
			// leave d.last untouched so the status panel keeps showing the
			// previous result.
			if !errors.Is(err, ErrNoSlaves) {
				status := "success"
				if errors.Is(err, ErrPartialFailure) {
					status = "partial_failure"
				} else if err != nil {
					status = "error"
				}
				d.last = &RunStatus{
					RunID:       req.runID,
					TriggeredBy: req.triggeredBy,
					StartedAt:   now,
					FinishedAt:  &fin,
					Status:      status,
				}
			}
			d.current = nil
			d.mu.Unlock()
		}
	}
}

// Submit queues a sync request. Returns the pre-generated run ID, or:
//   - ErrSyncBusy if the queue is full (another run is already queued or in progress)
//   - ErrNoSlaves if there are no enabled slave instances to sync to
func (d *Dispatcher) Submit(ctx context.Context, triggeredBy string) (string, error) {
	ok, err := d.engine.HasEnabledSlaves(ctx)
	if err != nil {
		return "", fmt.Errorf("pre-flight check: %w", err)
	}
	if !ok {
		return "", ErrNoSlaves
	}
	runID := uuid.NewString()
	req := dispatchReq{runID: runID, triggeredBy: triggeredBy}
	select {
	case d.queue <- req:
		return runID, nil
	default:
		return "", ErrSyncBusy
	}
}

// Status returns the current (running) and last (completed) run status.
// Either may be nil if no run has started yet.
// RunStatus values are never mutated after being stored — new structs are always
// allocated on each state transition — so a caller holding the returned pointer
// after the lock is released sees a consistent, immutable snapshot.
func (d *Dispatcher) Status() (current, last *RunStatus) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.current, d.last
}
