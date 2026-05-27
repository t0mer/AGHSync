package notification

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a channel does not exist.
var ErrNotFound = errors.New("notification channel not found")

// ErrDuplicateName is returned when a channel with the same name already exists.
var ErrDuplicateName = errors.New("a notification channel with this name already exists")

// ChannelType identifies the notification provider.
type ChannelType string

const (
	TypeShoutrrr ChannelType = "shoutrrr"
	TypeGreenAPI ChannelType = "greenapi"
	TypeWhatsApp ChannelType = "whatsapp"
)

// Channel is a persisted notification channel.
type Channel struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Type          ChannelType `json:"type"`
	Config        string      `json:"config"` // decrypted JSON, only populated on reads
	NotifySuccess bool        `json:"notify_success"`
	NotifyFailure bool        `json:"notify_failure"`
	Enabled       bool        `json:"enabled"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// ShoutrrrConfig holds the provider-specific fields for a Shoutrrr channel.
type ShoutrrrConfig struct {
	URL string `json:"url"`
}

// GreenAPIConfig holds the provider-specific fields for a GreenAPI channel.
type GreenAPIConfig struct {
	InstanceID string `json:"instance_id"`
	APIToken   string `json:"api_token"`
	Phone      string `json:"phone"`
}

// WhatsAppConfig holds the provider-specific fields for a go-whatsapp-web-multidevice channel.
type WhatsAppConfig struct {
	APIURL string `json:"api_url"`
	Phone  string `json:"phone"`
}
