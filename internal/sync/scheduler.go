package sync

import (
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler triggers syncs on a cron schedule via the Dispatcher.
type Scheduler struct {
	dispatcher *Dispatcher
	cron       *cron.Cron
	mu         sync.Mutex
	entryID    cron.EntryID
	hasEntry   bool
}

// NewScheduler creates a Scheduler backed by the given Dispatcher.
func NewScheduler(dispatcher *Dispatcher) *Scheduler {
	return &Scheduler{
		dispatcher: dispatcher,
		cron:       cron.New(cron.WithLocation(time.UTC)),
	}
}

// SetSchedule installs (or replaces) the cron expression.
// Pass "" to disable the schedule without error.
func (s *Scheduler) SetSchedule(expr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasEntry {
		s.cron.Remove(s.entryID)
		s.hasEntry = false
	}

	if expr == "" {
		return nil
	}

	// Validate the expression before adding.
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := p.Parse(expr); err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}

	id, err := s.cron.AddFunc(expr, func() {
		// Submit returns ErrSyncBusy when a sync is already queued/running; that
		// is expected and silently skipped — the cron will retry on the next tick.
		_, _ = s.dispatcher.Submit("scheduler")
	})
	if err != nil {
		return fmt.Errorf("add cron job: %w", err)
	}
	s.entryID = id
	s.hasEntry = true
	return nil
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() { s.cron.Start() }

// Stop stops the cron scheduler.
func (s *Scheduler) Stop() { s.cron.Stop() }
