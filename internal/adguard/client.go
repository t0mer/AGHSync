package adguard

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to a single AdGuardHome instance over HTTP.
type Client struct {
	base string
	user string
	pass string
	http *http.Client
}

// NewClient creates an AdGuardHome HTTP client.
func NewClient(address, username, password string, tlsSkipVerify bool) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: tlsSkipVerify}, //nolint:gosec
	}
	return &Client{
		base: address,
		user: username,
		pass: password,
		http: &http.Client{Transport: tr},
	}
}

// Snapshot reads the current configuration for configType from the AGH instance.
func (c *Client) Snapshot(ctx context.Context, configType string) (json.RawMessage, error) {
	switch configType {
	case "dns":
		return c.get(ctx, "/control/dns_info")
	case "dhcp":
		return c.get(ctx, "/control/dhcp/status")
	case "filtering":
		return c.get(ctx, "/control/filtering/status")
	case "blocked_services":
		return c.get(ctx, "/control/blocked_services/get")
	case "clients":
		return c.get(ctx, "/control/clients")
	case "rewrite":
		return c.rewriteSnapshot(ctx)
	case "safebrowsing":
		return c.get(ctx, "/control/safebrowsing/status")
	case "parental":
		return c.get(ctx, "/control/parental/status")
	case "safesearch":
		return c.get(ctx, "/control/safesearch/status")
	case "stats":
		return c.get(ctx, "/control/stats_info")
	case "tls":
		return c.get(ctx, "/control/tls/status")
	case "log":
		return c.get(ctx, "/control/querylog_info")
	default:
		return nil, fmt.Errorf("unknown config type: %s", configType)
	}
}

// Apply pushes a configuration snapshot to the AGH instance.
func (c *Client) Apply(ctx context.Context, configType string, data json.RawMessage) error {
	switch configType {
	case "dns":
		return c.post(ctx, "/control/dns_config", data)
	case "dhcp":
		return c.post(ctx, "/control/dhcp/set_config", data)
	case "filtering":
		return c.applyFiltering(ctx, data)
	case "blocked_services":
		return c.put(ctx, "/control/blocked_services/update", data)
	case "clients":
		return c.applyClients(ctx, data)
	case "rewrite":
		return c.applyRewrite(ctx, data)
	case "safebrowsing":
		return c.applyToggle(ctx, data, "/control/safebrowsing/enable", "/control/safebrowsing/disable")
	case "parental":
		return c.applyToggle(ctx, data, "/control/parental/enable", "/control/parental/disable")
	case "safesearch":
		return c.put(ctx, "/control/safesearch/settings", data)
	case "stats":
		return c.post(ctx, "/control/stats_config", data)
	case "tls":
		return c.post(ctx, "/control/tls/configure", data)
	case "log":
		return c.post(ctx, "/control/querylog_config", data)
	default:
		return fmt.Errorf("unknown config type: %s", configType)
	}
}

// TestConnection verifies connectivity and credentials using the same Basic Auth
// mechanism that all other API calls use.
func (c *Client) TestConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/control/status", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.user, c.pass)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("could not connect to %s: %w", c.base, err)
	}
	defer func() { io.Copy(io.Discard, resp.Body); resp.Body.Close() }() //nolint:errcheck
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("invalid username or password")
	case http.StatusTooManyRequests:
		return fmt.Errorf("too many login attempts — try again later")
	default:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("unexpected response: %d %s", resp.StatusCode, b)
	}
}

func (c *Client) applyFiltering(ctx context.Context, data json.RawMessage) error {
	var status struct {
		Enabled   bool     `json:"enabled"`
		Interval  float64  `json:"interval"`
		UserRules []string `json:"user_rules"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return fmt.Errorf("parse filtering snapshot: %w", err)
	}
	cfgBody, _ := json.Marshal(map[string]any{"enabled": status.Enabled, "interval": status.Interval})
	if err := c.post(ctx, "/control/filtering/config", cfgBody); err != nil {
		return fmt.Errorf("filtering/config: %w", err)
	}
	rules := status.UserRules
	if rules == nil {
		rules = []string{}
	}
	rulesBody, _ := json.Marshal(map[string]any{"rules": rules})
	if err := c.post(ctx, "/control/filtering/set_rules", rulesBody); err != nil {
		return fmt.Errorf("filtering/set_rules: %w", err)
	}
	return nil
}

func (c *Client) applyToggle(ctx context.Context, data json.RawMessage, enableURL, disableURL string) error {
	var s struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parse toggle snapshot: %w", err)
	}
	if s.Enabled {
		return c.post(ctx, enableURL, nil)
	}
	return c.post(ctx, disableURL, nil)
}

func (c *Client) applyClients(ctx context.Context, masterData json.RawMessage) error {
	var masterResp struct {
		Clients []json.RawMessage `json:"clients"`
	}
	if err := json.Unmarshal(masterData, &masterResp); err != nil {
		return fmt.Errorf("parse master clients: %w", err)
	}

	childRaw, err := c.get(ctx, "/control/clients")
	if err != nil {
		return fmt.Errorf("get child clients: %w", err)
	}
	var childResp struct {
		Clients []json.RawMessage `json:"clients"`
	}
	if err := json.Unmarshal(childRaw, &childResp); err != nil {
		return fmt.Errorf("parse child clients: %w", err)
	}

	type nameOnly struct{ Name string `json:"name"` }

	masterByName := make(map[string]json.RawMessage)
	for _, mc := range masterResp.Clients {
		var obj nameOnly
		if err := json.Unmarshal(mc, &obj); err == nil && obj.Name != "" {
			masterByName[obj.Name] = mc
		}
	}
	childByName := make(map[string]json.RawMessage)
	for _, cc := range childResp.Clients {
		var obj nameOnly
		if err := json.Unmarshal(cc, &obj); err == nil && obj.Name != "" {
			childByName[obj.Name] = cc
		}
	}

	for name, mc := range masterByName {
		if cc, exists := childByName[name]; !exists {
			body, _ := json.Marshal(map[string]json.RawMessage{"data": mc})
			if err := c.post(ctx, "/control/clients/add", body); err != nil {
				return fmt.Errorf("add client %q: %w", name, err)
			}
		} else if string(mc) != string(cc) {
			body, _ := json.Marshal(map[string]any{"name": name, "data": json.RawMessage(mc)})
			if err := c.post(ctx, "/control/clients/update", body); err != nil {
				return fmt.Errorf("update client %q: %w", name, err)
			}
		}
	}
	for name := range childByName {
		if _, exists := masterByName[name]; !exists {
			body, _ := json.Marshal(map[string]string{"name": name})
			if err := c.post(ctx, "/control/clients/delete", body); err != nil {
				return fmt.Errorf("delete client %q: %w", name, err)
			}
		}
	}
	return nil
}

func (c *Client) rewriteSnapshot(ctx context.Context) (json.RawMessage, error) {
	list, err := c.get(ctx, "/control/rewrite/list")
	if err != nil {
		return nil, err
	}
	settings, err := c.get(ctx, "/control/rewrite/settings")
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]json.RawMessage{"list": list, "settings": settings})
}

func (c *Client) applyRewrite(ctx context.Context, data json.RawMessage) error {
	var snap struct {
		List     json.RawMessage `json:"list"`
		Settings json.RawMessage `json:"settings"`
	}
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("parse rewrite snapshot: %w", err)
	}

	if err := c.put(ctx, "/control/rewrite/settings/update", snap.Settings); err != nil {
		return fmt.Errorf("rewrite/settings/update: %w", err)
	}

	type entry struct {
		Domain string `json:"domain"`
		Answer string `json:"answer"`
	}

	var masterList, childList []entry
	if err := json.Unmarshal(snap.List, &masterList); err != nil {
		return err
	}
	childRaw, err := c.get(ctx, "/control/rewrite/list")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(childRaw, &childList); err != nil {
		return err
	}

	masterSet := make(map[entry]bool)
	for _, e := range masterList {
		masterSet[e] = true
	}
	childSet := make(map[entry]bool)
	for _, e := range childList {
		childSet[e] = true
	}

	for k := range masterSet {
		if !childSet[k] {
			body, _ := json.Marshal(k)
			if err := c.post(ctx, "/control/rewrite/add", body); err != nil {
				return fmt.Errorf("rewrite/add %v: %w", k, err)
			}
		}
	}
	for k := range childSet {
		if !masterSet[k] {
			body, _ := json.Marshal(k)
			if err := c.post(ctx, "/control/rewrite/delete", body); err != nil {
				return fmt.Errorf("rewrite/delete %v: %w", k, err)
			}
		}
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.user, c.pass)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %d %s", path, resp.StatusCode, body)
	}
	return json.RawMessage(body), nil
}

func (c *Client) post(ctx context.Context, path string, body json.RawMessage) error {
	return c.doJSON(ctx, http.MethodPost, path, body)
}

func (c *Client) put(ctx context.Context, path string, body json.RawMessage) error {
	return c.doJSON(ctx, http.MethodPut, path, body)
}

func (c *Client) doJSON(ctx context.Context, method, path string, body json.RawMessage) error {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bodyReader)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.user, c.pass)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, b)
	}
	return nil
}
