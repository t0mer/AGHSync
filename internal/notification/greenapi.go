package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type greenAPISender struct {
	cfg GreenAPIConfig
}

func newGreenAPISender(configJSON string) (*greenAPISender, error) {
	var cfg GreenAPIConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("greenapi: parse config: %w", err)
	}
	if cfg.InstanceID == "" || cfg.APIToken == "" || cfg.Phone == "" {
		return nil, fmt.Errorf("greenapi: instance_id, api_token, and phone are required")
	}
	return &greenAPISender{cfg: cfg}, nil
}

func (s *greenAPISender) Send(ctx context.Context, message string) error {
	// POST https://api.green-api.com/waInstance{idInstance}/sendMessage/{apiTokenInstance}
	// chatId for a personal WhatsApp chat: {phone}@c.us  (international number, no + or spaces)
	endpoint := fmt.Sprintf("https://api.green-api.com/waInstance%s/sendMessage/%s", s.cfg.InstanceID, s.cfg.APIToken)
	payload, _ := json.Marshal(map[string]string{
		"chatId":  s.cfg.Phone + "@c.us",
		"message": message,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("greenapi: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("greenapi: send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		// Read up to 512 bytes of the error body so the caller can diagnose misconfiguration.
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("greenapi: status %d: %s", resp.StatusCode, string(errBody))
	}
	return nil
}
