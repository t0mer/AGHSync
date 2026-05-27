package adguard_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/adguard"
)

func newTestServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, h := range handlers {
		mux.HandleFunc(path, h)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestClient_Snapshot_DNS(t *testing.T) {
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/control/dns_info": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			user, pass, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "admin", user)
			assert.Equal(t, "secret", pass)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"upstream_dns":["8.8.8.8"]}`))
		},
	})

	c := adguard.NewClient(srv.URL, "admin", "secret", false)
	data, err := c.Snapshot(context.Background(), "dns")
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Contains(t, m, "upstream_dns")
}

func TestClient_Apply_DNS(t *testing.T) {
	var received []byte
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/control/dns_config": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			received, _ = json.Marshal(r.Body)
			w.WriteHeader(http.StatusOK)
		},
	})

	c := adguard.NewClient(srv.URL, "admin", "secret", false)
	err := c.Apply(context.Background(), "dns", json.RawMessage(`{"upstream_dns":["1.1.1.1"]}`))
	require.NoError(t, err)
	_ = received
}

func TestClient_Apply_DNS_DisablesPrivatePtrWhenNoUpstreams(t *testing.T) {
	var body map[string]any
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/control/dns_config": func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusOK)
		},
	})

	c := adguard.NewClient(srv.URL, "u", "p", false)
	// Snapshot has use_private_ptr_resolvers=true but empty local_ptr_upstreams.
	err := c.Apply(context.Background(), "dns", json.RawMessage(
		`{"upstream_dns":["1.1.1.1"],"use_private_ptr_resolvers":true,"local_ptr_upstreams":[]}`,
	))
	require.NoError(t, err)
	// Flag must be cleared to avoid AGH validation error.
	assert.Equal(t, false, body["use_private_ptr_resolvers"])
}

func TestClient_Apply_DNS_KeepsPrivatePtrWhenUpstreamsPresent(t *testing.T) {
	var body map[string]any
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/control/dns_config": func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusOK)
		},
	})

	c := adguard.NewClient(srv.URL, "u", "p", false)
	err := c.Apply(context.Background(), "dns", json.RawMessage(
		`{"upstream_dns":["1.1.1.1"],"use_private_ptr_resolvers":true,"local_ptr_upstreams":["192.168.1.1"]}`,
	))
	require.NoError(t, err)
	assert.Equal(t, true, body["use_private_ptr_resolvers"])
}

func TestClient_Snapshot_Rewrite_CombinesListAndSettings(t *testing.T) {
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/control/rewrite/list": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`[{"domain":"test.local","answer":"192.168.1.1"}]`))
		},
		"/control/rewrite/settings": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"some_setting":true}`))
		},
	})

	c := adguard.NewClient(srv.URL, "admin", "pass", false)
	data, err := c.Snapshot(context.Background(), "rewrite")
	require.NoError(t, err)

	var snap struct {
		List     json.RawMessage `json:"list"`
		Settings json.RawMessage `json:"settings"`
	}
	require.NoError(t, json.Unmarshal(data, &snap))
	assert.NotEmpty(t, snap.List)
	assert.NotEmpty(t, snap.Settings)
}

func TestClient_Apply_Filtering_SendsConfigAndRules(t *testing.T) {
	var configCalled, rulesCalled bool
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/control/filtering/config": func(w http.ResponseWriter, r *http.Request) {
			configCalled = true
			w.WriteHeader(http.StatusOK)
		},
		"/control/filtering/set_rules": func(w http.ResponseWriter, r *http.Request) {
			rulesCalled = true
			w.WriteHeader(http.StatusOK)
		},
	})

	c := adguard.NewClient(srv.URL, "admin", "pass", false)
	snap := json.RawMessage(`{"enabled":true,"interval":24,"user_rules":["||ads.com^"]}`)
	err := c.Apply(context.Background(), "filtering", snap)
	require.NoError(t, err)
	assert.True(t, configCalled, "filtering/config must be called")
	assert.True(t, rulesCalled, "filtering/set_rules must be called")
}

func TestClient_Apply_SafeBrowsing_Enable(t *testing.T) {
	var enableCalled bool
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/control/safebrowsing/enable": func(w http.ResponseWriter, r *http.Request) {
			enableCalled = true
			w.WriteHeader(http.StatusOK)
		},
	})

	c := adguard.NewClient(srv.URL, "admin", "pass", false)
	err := c.Apply(context.Background(), "safebrowsing", json.RawMessage(`{"enabled":true}`))
	require.NoError(t, err)
	assert.True(t, enableCalled)
}


func TestClient_UnknownConfigType(t *testing.T) {
	c := adguard.NewClient("http://localhost", "u", "p", false)
	_, err := c.Snapshot(context.Background(), "invalid_type")
	assert.Error(t, err)

	err = c.Apply(context.Background(), "invalid_type", nil)
	assert.Error(t, err)
}

func TestClient_TestConnection_Success(t *testing.T) {
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/control/status": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			user, pass, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "admin", user)
			assert.Equal(t, "secret", pass)
			w.WriteHeader(http.StatusOK)
		},
	})
	c := adguard.NewClient(srv.URL, "admin", "secret", false)
	assert.NoError(t, c.TestConnection(context.Background()))
}

func TestClient_TestConnection_BadCredentials(t *testing.T) {
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/control/status": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		},
	})
	c := adguard.NewClient(srv.URL, "admin", "wrong", false)
	err := c.TestConnection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid username or password")
}

func TestClient_TestConnection_RateLimited(t *testing.T) {
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/control/status": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		},
	})
	c := adguard.NewClient(srv.URL, "admin", "pass", false)
	err := c.TestConnection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many login attempts")
}

func TestClient_TestConnection_Unreachable(t *testing.T) {
	c := adguard.NewClient("http://127.0.0.1:19999", "u", "p", false)
	err := c.TestConnection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not connect to")
}
