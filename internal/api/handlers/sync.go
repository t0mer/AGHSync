package handlers

import (
	"errors"
	"net/http"

	"github.com/t0mer/aghsync/internal/config"
	internalsync "github.com/t0mer/aghsync/internal/sync"
)

// TriggerSync handles POST /sync/run. Returns 202 with the queued run ID.
func TriggerSync(d *internalsync.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runID, err := d.Submit("manual")
		if err != nil {
			if errors.Is(err, internalsync.ErrSyncBusy) {
				WriteError(w, http.StatusConflict, "sync already in progress")
				return
			}
			WriteError(w, http.StatusInternalServerError, "submit failed")
			return
		}
		WriteJSON(w, http.StatusAccepted, map[string]string{"run_id": runID})
	}
}

// GetSyncStatus handles GET /sync/status.
func GetSyncStatus(d *internalsync.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, last := d.Status()
		WriteJSON(w, http.StatusOK, map[string]any{
			"current": current,
			"last":    last,
		})
	}
}

type scheduleRequest struct {
	Cron string `json:"cron"`
}

// SetSchedule handles PUT /sync/schedule.
func SetSchedule(cfg *config.Config, sched *internalsync.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req scheduleRequest
		if err := DecodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := sched.SetSchedule(req.Cron); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid cron expression")
			return
		}
		if err := cfg.SetSchedulerCron(req.Cron); err != nil {
			WriteError(w, http.StatusInternalServerError, "save schedule failed")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"cron": req.Cron})
	}
}
