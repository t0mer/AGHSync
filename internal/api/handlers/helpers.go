package handlers

import (
	"encoding/json"
	"io"
	"net/http"
)

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// DecodeJSON decodes the request body into dst.
// Returns an error for unknown fields or malformed JSON.
// Body reads are limited to 1 MB to prevent unbounded memory consumption.
func DecodeJSON(r *http.Request, dst any) error {
	r.Body = io.NopCloser(io.LimitReader(r.Body, 1<<20))
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
