package jphttpvaultx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers shared across vault_test.go and secrets_test.go
// ---------------------------------------------------------------------------

// newTestClient wires a Client to the provided httptest.Server.
func newTestClient(srv *httptest.Server, kvMount, namespace string) *Client {
	return New(
		WithProxyAddr(srv.URL),
		WithKVMount(kvMount),
		WithNamespace(namespace),
		WithTimeout(2*time.Second),
	)
}

// vaultKVResponse builds the standard Vault KV v2 success envelope:
//
//	{ "data": { "data": <innerData>, "metadata": {} } }
func vaultKVResponse(innerData map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"data":     innerData,
			"metadata": map[string]interface{}{},
		},
	}
}

// writeJSON encodes v as JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ---------------------------------------------------------------------------
// New / options
// ---------------------------------------------------------------------------

func TestNewDefaults(t *testing.T) {
	c := New()
	assert.Equal(t, defaultProxyAddr, c.proxyAddr)
	assert.Equal(t, defaultKVMount, c.kvMount)
	assert.Empty(t, c.namespace)
	assert.NotNil(t, c.httpClient)
}

func TestNewWithOptions(t *testing.T) {
	c := New(
		WithProxyAddr("http://10.0.0.1:8200"),
		WithKVMount("my-mount"),
		WithNamespace("my-ns"),
		WithTimeout(10*time.Second),
	)
	assert.Equal(t, "http://10.0.0.1:8200", c.proxyAddr)
	assert.Equal(t, "my-mount", c.kvMount)
	assert.Equal(t, "my-ns", c.namespace)
	assert.Equal(t, 10*time.Second, c.httpClient.Timeout)
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 42 * time.Second}
	c := New(WithHTTPClient(custom))
	assert.Same(t, custom, c.httpClient)
}

// ---------------------------------------------------------------------------
// do — namespace header
// ---------------------------------------------------------------------------

func TestDoNamespaceHeader(t *testing.T) {
	var receivedNS string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedNS = r.Header.Get("X-Vault-Namespace")
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": map[string]interface{}{}})
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "corp/team")
	_, _ = c.get(context.Background(), "secret/data/foo")

	assert.Equal(t, "corp/team", receivedNS)
}

func TestDoNoNamespaceHeader(t *testing.T) {
	var receivedNS string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedNS = r.Header.Get("X-Vault-Namespace")
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": map[string]interface{}{}})
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, _ = c.get(context.Background(), "secret/data/foo")

	assert.Empty(t, receivedNS)
}

// ---------------------------------------------------------------------------
// do — HTTP status handling
// ---------------------------------------------------------------------------

func TestDo204NoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	data, err := c.get(context.Background(), "some/path")
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestDo400Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"errors": []string{"permission denied"},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.get(context.Background(), "some/path")
	require.Error(t, err)
}

func TestDo404Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"errors": []string{}})
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.get(context.Background(), "missing/path")
	require.Error(t, err)
}

func TestDoInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.get(context.Background(), "some/path")
	require.Error(t, err)
}

func TestDoCancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": map[string]interface{}{}})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	c := newTestClient(srv, "secret", "")
	_, err := c.get(ctx, "some/path")
	require.Error(t, err)
}
