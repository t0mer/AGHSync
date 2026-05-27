package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type whatsAppSender struct {
	cfg    WhatsAppConfig
	client *http.Client
}

func newWhatsAppSender(configJSON string) (*whatsAppSender, error) {
	var cfg WhatsAppConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("whatsapp: parse config: %w", err)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Recipient) == "" {
		return nil, fmt.Errorf("whatsapp: base_url and recipient are required")
	}
	return &whatsAppSender{cfg: cfg, client: defaultHTTPClient()}, nil
}

func (s *whatsAppSender) Send(ctx context.Context, message string) error {
	endpoint := strings.TrimRight(s.cfg.BaseURL, "/") + "/send/message"
	payload, _ := json.Marshal(map[string]string{"phone": s.cfg.Recipient, "message": message})

	var auth func(*http.Request)
	if user := strings.TrimSpace(s.cfg.Username); user != "" {
		user, pass := user, s.cfg.Password
		auth = func(req *http.Request) { req.SetBasicAuth(user, pass) }
	}

	return postJSON(ctx, s.client, endpoint, payload, auth, "whatsapp")
}
