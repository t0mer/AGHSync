package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const greenAPIDefaultBase = "https://api.green-api.com"

type greenAPISender struct {
	cfg    GreenAPIConfig
	client *http.Client
}

func newGreenAPISender(configJSON string) (*greenAPISender, error) {
	var cfg GreenAPIConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("greenapi: parse config: %w", err)
	}
	if strings.TrimSpace(cfg.InstanceID) == "" || strings.TrimSpace(cfg.Token) == "" || strings.TrimSpace(cfg.Recipient) == "" {
		return nil, fmt.Errorf("greenapi: instance_id, token, and recipient are required")
	}
	return &greenAPISender{cfg: cfg, client: defaultHTTPClient()}, nil
}

func (s *greenAPISender) Send(ctx context.Context, message string) error {
	base := strings.TrimSpace(s.cfg.APIURL)
	if base == "" {
		base = greenAPIDefaultBase
	}

	chatID := strings.TrimSpace(s.cfg.Recipient)
	if !strings.Contains(chatID, "@") {
		chatID += "@c.us"
	}

	endpoint := fmt.Sprintf("%s/waInstance%s/sendMessage/%s",
		strings.TrimRight(base, "/"), s.cfg.InstanceID, s.cfg.Token)
	payload, _ := json.Marshal(map[string]string{"chatId": chatID, "message": message})

	return postJSON(ctx, s.client, endpoint, payload, nil, "greenapi")
}
