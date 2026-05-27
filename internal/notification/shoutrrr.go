package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/containrrr/shoutrrr"
)

type shoutrrrSender struct {
	url string
}

func newShoutrrrSender(configJSON string) (*shoutrrrSender, error) {
	var cfg ShoutrrrConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("shoutrrr: parse config: %w", err)
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("shoutrrr: url is required")
	}
	return &shoutrrrSender{url: cfg.URL}, nil
}

func (s *shoutrrrSender) Send(_ context.Context, message string) error {
	if err := shoutrrr.Send(s.url, message); err != nil {
		return fmt.Errorf("shoutrrr send: %w", err)
	}
	return nil
}
