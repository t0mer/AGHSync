package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/t0mer/aghsync/internal/adguard"
	"github.com/t0mer/aghsync/internal/history"
	"github.com/t0mer/aghsync/internal/instance"
	"github.com/t0mer/aghsync/internal/notification"
)

// ErrNoMaster is returned when no master instance is configured.
var ErrNoMaster = errors.New("no master instance configured")

// ErrNoSlaves is returned when there are no enabled slave instances to sync to.
var ErrNoSlaves = errors.New("no enabled slave instances to sync to")

// ErrPartialFailure is returned by Run when at least one child sync failed.
var ErrPartialFailure = errors.New("partial failure: one or more child instances failed")

// Engine runs a full configuration sync from master to all child instances.
type Engine struct {
	instances    *instance.Repository
	history      *history.Store
	notifier     *notification.Service
	logger       *slog.Logger
}

// NewEngine creates a sync Engine.
func NewEngine(instances *instance.Repository, hist *history.Store) *Engine {
	return &Engine{instances: instances, history: hist, logger: slog.Default()}
}

// NewEngineWithLogger creates a sync Engine with a specific logger.
func NewEngineWithLogger(instances *instance.Repository, hist *history.Store, logger *slog.Logger) *Engine {
	return &Engine{instances: instances, history: hist, logger: logger}
}

// WithNotifier attaches a notification service to the engine.
func (e *Engine) WithNotifier(n *notification.Service) *Engine {
	e.notifier = n
	return e
}

// HasEnabledSlaves returns true if at least one enabled (non-master) instance exists.
func (e *Engine) HasEnabledSlaves(ctx context.Context) (bool, error) {
	instances, err := e.instances.List(ctx)
	if err != nil {
		return false, err
	}
	for _, inst := range instances {
		if !inst.IsMaster && inst.SyncEnabled {
			return true, nil
		}
	}
	return false, nil
}

// Run executes a full sync cycle. It records progress in the history store using runID.
// Returns ErrNoSlaves (without writing any history) if there are no enabled slave instances.
func (e *Engine) Run(ctx context.Context, runID, triggeredBy string) error {
	// Pre-flight: skip history entirely when there is nothing to sync to.
	instances, err := e.instances.List(ctx)
	if err != nil {
		return fmt.Errorf("list instances: %w", err)
	}
	hasSlaves := false
	for _, inst := range instances {
		if !inst.IsMaster && inst.SyncEnabled {
			hasSlaves = true
			break
		}
	}
	if !hasSlaves {
		return ErrNoSlaves
	}

	if _, err := e.history.StartRun(ctx, runID, triggeredBy); err != nil {
		return fmt.Errorf("start run: %w", err)
	}

	finalStatus, err := e.doSync(ctx, runID)
	finCtx := context.Background()
	if err != nil {
		_ = e.history.FinishRun(finCtx, runID, "error")
		e.sendNotification(finCtx, runID, "error")
		return err
	}
	if finishErr := e.history.FinishRun(finCtx, runID, finalStatus); finishErr != nil {
		return finishErr
	}
	e.sendNotification(finCtx, runID, finalStatus)
	if finalStatus == "partial_failure" {
		return ErrPartialFailure
	}
	return nil
}

func (e *Engine) doSync(ctx context.Context, runID string) (string, error) {
	instances, err := e.instances.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list instances: %w", err)
	}

	var master *instance.Instance
	var children []*instance.Instance
	for _, inst := range instances {
		if inst.IsMaster {
			master = inst
		} else if inst.SyncEnabled {
			children = append(children, inst)
		}
	}
	if master == nil {
		return "", ErrNoMaster
	}

	masterPw, err := e.instances.GetDecryptedPassword(ctx, master.ID)
	if err != nil {
		return "", fmt.Errorf("get master credentials: %w", err)
	}
	masterClient := adguard.NewClient(master.Address, master.Username, masterPw, master.TLSSkipVerify)

	// Determine which config types are enabled on the master.
	syncConf, err := e.instances.GetSyncConfig(ctx, master.ID)
	if err != nil {
		return "", fmt.Errorf("get master sync config: %w", err)
	}
	enabledTypes := make(map[string]bool)
	for _, sc := range syncConf {
		if sc.Enabled {
			enabledTypes[sc.ConfigType] = true
		}
	}

	// Take snapshots from master for enabled config types only.
	snapshots := make(map[string]json.RawMessage)
	for _, ct := range instance.AllConfigTypes {
		if !enabledTypes[ct] {
			continue
		}
		snap, err := masterClient.Snapshot(ctx, ct)
		if err != nil {
			e.logger.Warn("master snapshot failed", "config_type", ct, "err", err)
			continue
		}
		snapshots[ct] = snap
	}

	if len(children) == 0 {
		return "success", nil
	}

	// Fan-out to children; at most 5 concurrent.
	sem := make(chan struct{}, 5)
	type childResult struct{ hasError bool }
	resultsCh := make(chan childResult, len(children))

	var wg sync.WaitGroup
	for _, child := range children {
		wg.Add(1)
		go func(child *instance.Instance) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			hasErr := e.syncChild(ctx, runID, child, snapshots)
			resultsCh <- childResult{hasError: hasErr}
		}(child)
	}
	wg.Wait()
	close(resultsCh)

	anyError := false
	for r := range resultsCh {
		if r.hasError {
			anyError = true
		}
	}
	if anyError {
		return "partial_failure", nil
	}
	return "success", nil
}

// sendNotification fetches the finished run and its results then delegates to the
// notification service. It is a no-op when no notifier is configured.
func (e *Engine) sendNotification(ctx context.Context, runID, status string) {
	if e.notifier == nil {
		return
	}
	run, err := e.history.GetRun(ctx, runID)
	if err != nil {
		e.logger.Warn("notification: fetch run failed", "run", runID, "err", err)
		return
	}
	results, err := e.history.GetResults(ctx, runID)
	if err != nil {
		e.logger.Warn("notification: fetch results failed", "run", runID, "err", err)
		results = nil
	}
	_ = status // run.Status is already set by FinishRun
	e.notifier.Notify(ctx, run, results)
}

// syncChild applies master snapshots to one child instance.
// Returns true if any config type failed.
func (e *Engine) syncChild(ctx context.Context, runID string, child *instance.Instance, snapshots map[string]json.RawMessage) bool {
	childPw, err := e.instances.GetDecryptedPassword(ctx, child.ID)
	if err != nil {
		e.logger.Error("get child credentials", "instance", child.Name, "err", err)
		return true
	}

	childClient := adguard.NewClient(child.Address, child.Username, childPw, child.TLSSkipVerify)

	anyError := false
	for ct, masterSnap := range snapshots {
		// Capture child's state before applying (for diff).
		before, _ := childClient.Snapshot(ctx, ct)
		var diffJSON *string
		if before != nil {
			d := fmt.Sprintf(`{"before":%s,"after":%s}`, string(before), string(masterSnap))
			diffJSON = &d
		}

		applyErr := childClient.Apply(ctx, ct, masterSnap)
		if applyErr != nil {
			e.logger.Warn("apply failed", "instance", child.Name, "config_type", ct, "err", applyErr)
			msg := applyErr.Error()
			_ = e.history.AddResult(ctx, runID, child.ID, ct, "error", nil, &msg)
			anyError = true
			continue
		}
		_ = e.history.AddResult(ctx, runID, child.ID, ct, "success", diffJSON, nil)
	}
	return anyError
}
