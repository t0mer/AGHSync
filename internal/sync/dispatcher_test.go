package sync_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/history"
	"github.com/t0mer/aghsync/internal/instance"
	internalsync "github.com/t0mer/aghsync/internal/sync"
	"github.com/t0mer/aghsync/internal/store"
)

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestDispatcher(t *testing.T) *internalsync.Dispatcher {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	secret := make([]byte, 32)
	repo := instance.NewRepository(s.DB(), secret)
	hs := history.New(s.DB())
	engine := internalsync.NewEngineWithLogger(repo, hs, nopLogger())
	return internalsync.NewDispatcher(engine)
}

func TestDispatcher_Submit_ReturnsRunID(t *testing.T) {
	d := newTestDispatcher(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	runID, err := d.Submit("manual")
	require.NoError(t, err)
	assert.NotEmpty(t, runID)
}

func TestDispatcher_Submit_BusyReturnsSyncBusy(t *testing.T) {
	d := newTestDispatcher(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	// First submit should succeed.
	runID1, err := d.Submit("manual")
	require.NoError(t, err)
	assert.NotEmpty(t, runID1)

	// Second submit may or may not race; if it errors, it must be ErrSyncBusy.
	runID2, err2 := d.Submit("manual")
	if err2 != nil {
		assert.ErrorIs(t, err2, internalsync.ErrSyncBusy)
	} else {
		assert.NotEmpty(t, runID2)
	}
}

func TestDispatcher_Status_ReflectsLastRun(t *testing.T) {
	d := newTestDispatcher(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	runID, err := d.Submit("manual")
	require.NoError(t, err)

	// Wait for run to complete.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, last := d.Status()
		if last != nil && last.RunID == runID {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	_, last := d.Status()
	require.NotNil(t, last)
	assert.Equal(t, runID, last.RunID)
	assert.NotEqual(t, "running", last.Status)
}

func TestScheduler_SetSchedule_InvalidExprReturnsError(t *testing.T) {
	d := newTestDispatcher(t)
	sched := internalsync.NewScheduler(d)

	err := sched.SetSchedule("not-a-cron-expression")
	assert.Error(t, err)
}

func TestScheduler_SetSchedule_EmptyDisables(t *testing.T) {
	d := newTestDispatcher(t)
	sched := internalsync.NewScheduler(d)
	sched.Start()
	defer sched.Stop()

	require.NoError(t, sched.SetSchedule("* * * * *"))
	require.NoError(t, sched.SetSchedule("")) // disable — should not error
}
