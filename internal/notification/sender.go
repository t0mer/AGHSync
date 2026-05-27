package notification

import "context"

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
