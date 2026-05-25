package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/t0mer/aghsync/internal/history"
)

// ListHistory handles GET /history?limit=N&offset=N.
func ListHistory(hs *history.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		offset := 0
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}

		runs, err := hs.ListRuns(r.Context(), limit, offset)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "list runs failed")
			return
		}
		if runs == nil {
			runs = []*history.Run{}
		}
		WriteJSON(w, http.StatusOK, runs)
	}
}

type runWithResults struct {
	*history.Run
	Results []*history.Result `json:"results"`
}

// GetHistoryRun handles GET /history/{runId}.
func GetHistoryRun(hs *history.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runID := chi.URLParam(r, "runId")

		run, err := hs.GetRun(r.Context(), runID)
		if errors.Is(err, history.ErrRunNotFound) {
			WriteError(w, http.StatusNotFound, "run not found")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "get run failed")
			return
		}

		results, err := hs.GetResults(r.Context(), runID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "get results failed")
			return
		}
		if results == nil {
			results = []*history.Result{}
		}
		WriteJSON(w, http.StatusOK, runWithResults{run, results})
	}
}
