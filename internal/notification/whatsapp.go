package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type whatsAppSender struct {
	cfg WhatsAppConfig
}

func newWhatsAppSender(configJSON string) (*whatsAppSender, error) {
	var cfg WhatsAppConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("whatsapp: parse config: %w", err)
	}
	if cfg.APIURL == "" || cfg.Phone == "" {
		return nil, fmt.Errorf("whatsapp: api_url and phone are required")
	}
	return &whatsAppSender{cfg: cfg}, nil
}

func (s *whatsAppSender) Send(ctx context.Context, message string) error {
	// POST {api_url}/api/send/message
	url := strings.TrimRight(s.cfg.APIURL, "/") + "/api/send/message"
	body, _ := json.Marshal(map[string]string{
		"phone":   s.cfg.Phone,
		"message": message,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("whatsapp: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp: send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("whatsapp: unexpected status %d", resp.StatusCode)
	}
	return nil
}
