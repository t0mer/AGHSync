package instance

import "time"

// Instance represents an AdGuardHome server managed by AGHSync.
type Instance struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Address       string    `json:"address"`
	Username      string    `json:"username"`
	IsMaster      bool      `json:"is_master"`
	TLSSkipVerify bool      `json:"tls_skip_verify"`
	SyncEnabled   bool      `json:"sync_enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SyncConfigEntry represents one config-type toggle for a child instance.
type SyncConfigEntry struct {
	ConfigType string `json:"config_type"`
	Enabled    bool   `json:"enabled"`
}

// AllConfigTypes is the canonical ordered list of AGH config types that AGHSync
// can synchronise. Derived from the swagger.yml tag names.
var AllConfigTypes = []string{
	"blocked_services",
	"dhcp",
	"dns",
	"filtering",
	"parental",
	"rewrite",
	"safebrowsing",
	"safesearch",
	"tls",
}

// defaultEnabled returns the default enabled state for a config type.
// dhcp is off by default because DHCP configuration is host-specific and
// syncing it blindly across instances is almost always undesirable.
func defaultEnabled(configType string) int {
	if configType == "dhcp" {
		return 0
	}
	return 1
}
