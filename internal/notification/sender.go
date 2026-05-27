package notification

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Sender delivers a notification message to a single channel.
type Sender interface {
	Send(ctx context.Context, message string) error
}

// NewSender returns the correct Sender for the given channel.
func NewSender(ch *Channel) (Sender, error) {
	switch ch.Type {
	case TypeShoutrrr:
		return newShoutrrrSender(ch.Config)
	case TypeGreenAPI:
		return newGreenAPISender(ch.Config)
	case TypeWhatsApp:
		return newWhatsAppSender(ch.Config)
	default:
		return nil, ErrNotFound
	}
}

// defaultHTTPClient returns an http.Client with a 15-second timeout.
func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

// postJSON posts body as JSON to url, optionally applying auth to the request.
// Reads up to 512 bytes of an error response body for diagnostics.
func postJSON(ctx context.Context, client *http.Client, url string, body []byte, auth func(*http.Request), label string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: build request: %w", label, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if auth != nil {
		auth(req)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s: status %d: %s", label, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return nil
}
