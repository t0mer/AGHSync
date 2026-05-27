package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/t0mer/aghsync/internal/notification"
)

type channelRequest struct {
	Name          string                   `json:"name"`
	Type          notification.ChannelType `json:"type"`
	Config        string                   `json:"config"`
	NotifySuccess bool                     `json:"notify_success"`
	NotifyFailure bool                     `json:"notify_failure"`
	Enabled       bool                     `json:"enabled"`
}

type testChannelRequest struct {
	Type   notification.ChannelType `json:"type"`
	Config string                   `json:"config"`
}

// ListNotificationChannels returns all notification channels (configs included).
func ListNotificationChannels(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := repo.List(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to list channels")
			return
		}
		if list == nil {
			list = []*notification.Channel{}
		}
		WriteJSON(w, http.StatusOK, list)
	}
}

// CreateNotificationChannel inserts a new channel.
func CreateNotificationChannel(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req channelRequest
		if err := DecodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := validateChannelRequest(req); err != "" {
			WriteError(w, http.StatusBadRequest, err)
			return
		}
		ch, err := repo.Create(r.Context(), req.Name, req.Type, req.Config, req.NotifySuccess, req.NotifyFailure, req.Enabled)
		if err != nil {
			if err == notification.ErrDuplicateName {
				WriteError(w, http.StatusConflict, err.Error())
				return
			}
			WriteError(w, http.StatusInternalServerError, "failed to create channel")
			return
		}
		WriteJSON(w, http.StatusCreated, ch)
	}
}

// GetNotificationChannel returns a single channel by ID.
func GetNotificationChannel(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		ch, err := repo.Get(r.Context(), id)
		if err == notification.ErrNotFound {
			WriteError(w, http.StatusNotFound, "channel not found")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to get channel")
			return
		}
		WriteJSON(w, http.StatusOK, ch)
	}
}

// UpdateNotificationChannel modifies an existing channel.
// Pass config="" to keep the existing encrypted value.
func UpdateNotificationChannel(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req channelRequest
		if err := DecodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Config != "" {
			if err := validateChannelRequest(req); err != "" {
				WriteError(w, http.StatusBadRequest, err)
				return
			}
		}
		ch, err := repo.Update(r.Context(), id, req.Name, req.Type, req.Config, req.NotifySuccess, req.NotifyFailure, req.Enabled)
		if err == notification.ErrNotFound {
			WriteError(w, http.StatusNotFound, "channel not found")
			return
		}
		if err == notification.ErrDuplicateName {
			WriteError(w, http.StatusConflict, err.Error())
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to update channel")
			return
		}
		WriteJSON(w, http.StatusOK, ch)
	}
}

// DeleteNotificationChannel removes a channel by ID.
func DeleteNotificationChannel(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := repo.Delete(r.Context(), id)
		if err == notification.ErrNotFound {
			WriteError(w, http.StatusNotFound, "channel not found")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to delete channel")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// TestNotificationChannel sends a test message to verify channel config without saving.
func TestNotificationChannel(_ *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req testChannelRequest
		if err := DecodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Config == "" {
			WriteError(w, http.StatusBadRequest, "config is required")
			return
		}
		ch := &notification.Channel{Type: req.Type, Config: req.Config}
		sender, err := notification.NewSender(ch)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid channel config: "+err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		if err := sender.Send(ctx, "AGHSync test notification — channel is working."); err != nil {
			WriteError(w, http.StatusBadGateway, "send failed: "+err.Error())
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func validateChannelRequest(req channelRequest) string {
	if req.Name == "" {
		return "name is required"
	}
	switch req.Type {
	case notification.TypeShoutrrr, notification.TypeGreenAPI, notification.TypeWhatsApp:
	default:
		return "type must be shoutrrr, greenapi, or whatsapp"
	}
	if req.Config == "" {
		return "config is required"
	}
	return ""
}
