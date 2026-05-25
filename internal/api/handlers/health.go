package handlers

import (
	"encoding/json"
	"net/http"
)

// Health handles GET /api/v1/health.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
