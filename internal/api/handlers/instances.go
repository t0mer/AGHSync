package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/t0mer/aghsync/internal/adguard"
	"github.com/t0mer/aghsync/internal/instance"
)

type createInstanceRequest struct {
	Name          string `json:"name"`
	Address       string `json:"address"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	IsMaster      bool   `json:"is_master"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
}

type updateInstanceRequest struct {
	Name          string  `json:"name"`
	Address       string  `json:"address"`
	Username      string  `json:"username"`
	Password      *string `json:"password"`
	TLSSkipVerify bool    `json:"tls_skip_verify"`
}

type setSyncConfigRequest struct {
	Config []instance.SyncConfigEntry `json:"config"`
}

// ListInstances returns all instances.
func ListInstances(repo *instance.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := repo.List(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to list instances")
			return
		}
		if list == nil {
			list = []*instance.Instance{}
		}
		WriteJSON(w, http.StatusOK, list)
	}
}

// CreateInstance adds a new instance.
func CreateInstance(repo *instance.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createInstanceRequest
		if err := DecodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" {
			WriteError(w, http.StatusBadRequest, "name is required")
			return
		}
		if req.Address == "" {
			WriteError(w, http.StatusBadRequest, "address is required")
			return
		}
		inst, err := repo.Create(r.Context(), req.Name, req.Address, req.Username, req.Password, req.IsMaster, req.TLSSkipVerify)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to create instance")
			return
		}
		WriteJSON(w, http.StatusCreated, inst)
	}
}

// GetInstance returns a single instance by ID.
func GetInstance(repo *instance.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		inst, err := repo.Get(r.Context(), id)
		if errors.Is(err, instance.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "instance not found")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to get instance")
			return
		}
		WriteJSON(w, http.StatusOK, inst)
	}
}

// UpdateInstance modifies an existing instance.
func UpdateInstance(repo *instance.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req updateInstanceRequest
		if err := DecodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" {
			WriteError(w, http.StatusBadRequest, "name is required")
			return
		}
		if req.Address == "" {
			WriteError(w, http.StatusBadRequest, "address is required")
			return
		}
		inst, err := repo.Update(r.Context(), id, req.Name, req.Address, req.Username, req.Password, req.TLSSkipVerify)
		if errors.Is(err, instance.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "instance not found")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to update instance")
			return
		}
		WriteJSON(w, http.StatusOK, inst)
	}
}

// DeleteInstance removes an instance.
func DeleteInstance(repo *instance.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := repo.Delete(r.Context(), id)
		if errors.Is(err, instance.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "instance not found")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to delete instance")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// PromoteInstance promotes an instance to master.
func PromoteInstance(repo *instance.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := repo.Promote(r.Context(), id); err != nil {
			if errors.Is(err, instance.ErrNotFound) {
				WriteError(w, http.StatusNotFound, "instance not found")
				return
			}
			WriteError(w, http.StatusInternalServerError, "failed to promote instance")
			return
		}
		inst, err := repo.Get(r.Context(), id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to get promoted instance")
			return
		}
		WriteJSON(w, http.StatusOK, inst)
	}
}

// GetSyncConfig returns the sync config for an instance.
func GetSyncConfig(repo *instance.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		cfg, err := repo.GetSyncConfig(r.Context(), id)
		if errors.Is(err, instance.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "instance not found")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to get sync config")
			return
		}
		if cfg == nil {
			cfg = []instance.SyncConfigEntry{}
		}
		WriteJSON(w, http.StatusOK, cfg)
	}
}

// UpdateSyncConfig replaces the sync config for an instance.
func UpdateSyncConfig(repo *instance.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req setSyncConfigRequest
		if err := DecodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := repo.SetSyncConfig(r.Context(), id, req.Config); err != nil {
			if errors.Is(err, instance.ErrNotFound) {
				WriteError(w, http.StatusNotFound, "instance not found")
				return
			}
			WriteError(w, http.StatusInternalServerError, "failed to update sync config")
			return
		}
		cfg, _ := repo.GetSyncConfig(r.Context(), id)
		if cfg == nil {
			cfg = []instance.SyncConfigEntry{}
		}
		WriteJSON(w, http.StatusOK, cfg)
	}
}

type testConnectionRequest struct {
	Address       string `json:"address"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
}

func validateInstanceAddress(address string) error {
	u, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("address must use http or https scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("address must include a host")
	}
	return nil
}

// TestConnectionHandler tests connectivity to an AdGuardHome instance without saving it.
func TestConnectionHandler(w http.ResponseWriter, r *http.Request) {
	var req testConnectionRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Address == "" {
		WriteError(w, http.StatusBadRequest, "address is required")
		return
	}
	if err := validateInstanceAddress(req.Address); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	c := adguard.NewClient(req.Address, req.Username, req.Password, req.TLSSkipVerify)
	if err := c.TestConnection(r.Context()); err != nil {
		WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
