package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/t0mer/aghsync/internal/backup"
	"github.com/t0mer/aghsync/internal/config"
)

// ExportBackup streams the current application state as a downloadable JSON file.
func ExportBackup(db *sql.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := backup.Export(r.Context(), db, cfg)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, fmt.Sprintf("export failed: %s", err))
			return
		}
		filename := fmt.Sprintf("aghsync-backup-%s.json", time.Now().UTC().Format("2006-01-02"))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		w.WriteHeader(http.StatusOK)
		writeJSONRaw(w, data)
	}
}

// RestoreBackup replaces the application state with the contents of the uploaded backup.
func RestoreBackup(db *sql.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data backup.BackupData
		if err := DecodeJSONLarge(r, &data); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid backup file: "+err.Error())
			return
		}
		if err := backup.Restore(r.Context(), db, cfg, &data); err != nil {
			WriteError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
