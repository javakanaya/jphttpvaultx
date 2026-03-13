// Package jphttpvaultx provides a lightweight HTTP client for the
// HashiCorp Vault Lambda Extension proxy.
//
// The extension runs as a sidecar inside the Lambda execution environment
// and exposes a local HTTP proxy at http://127.0.0.1:8200. It handles
// authentication with Vault (via AWS IAM) transparently, so callers do
// not need to manage tokens.
//
// Usage:
//
//	client := jphttpvaultx.New(
//	    jphttpvaultx.WithKVMount("static-secret"),
//	    jphttpvaultx.WithNamespace("my-ns"), // Vault Enterprise only
//	)
//
//	secret, err := client.GetSecret(ctx, "my-path", "my-key")
package jphttpvaultx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultProxyAddr = "http://127.0.0.1:8200"
	defaultKVMount   = "secret"
	defaultTimeout   = 5 * time.Second

	// vaultURLFormat is the base pattern for all Vault v1 API endpoint URLs.
	vaultURLFormat = "%s/v1/%s"
)

// Client is a Vault Lambda Extension HTTP client.
// Use New() to construct one.
type Client struct {
	proxyAddr  string
	namespace  string
	kvMount    string
	httpClient *http.Client
}

// New creates a new Client with sensible defaults.
// Override defaults using functional options (WithProxyAddr, WithNamespace, etc.).
func New(opts ...Option) *Client {
	c := &Client{
		proxyAddr: defaultProxyAddr,
		kvMount:   defaultKVMount,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// vaultResponse mirrors the top-level Vault HTTP API response envelope.
type vaultResponse struct {
	Data   map[string]interface{} `json:"data"`
	Errors []string               `json:"errors"`
}

// get sends a GET request to the given Vault API path (relative to /v1/).
func (c *Client) get(ctx context.Context, path string) (map[string]interface{}, error) {
	url := fmt.Sprintf(vaultURLFormat, c.proxyAddr, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// post sends a POST request with a JSON body to the given Vault API path.
func (c *Client) post(ctx context.Context, path string, body any) (map[string]interface{}, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf(vaultURLFormat, c.proxyAddr, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

// delete sends a DELETE request to the given Vault API path.
func (c *Client) delete(ctx context.Context, path string) error {
	url := fmt.Sprintf(vaultURLFormat, c.proxyAddr, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	_, err = c.do(req)
	return err
}

// do executes a prepared HTTP request, injects common Vault headers,
// reads and decodes the response body.
func (c *Client) do(req *http.Request) (map[string]interface{}, error) {
	if c.namespace != "" {
		req.Header.Set("X-Vault-Namespace", c.namespace)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault proxy request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault response: %w", err)
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	var result vaultResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to decode vault response (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("vault returned %d: %v", resp.StatusCode, result.Errors)
	}

	return result.Data, nil
}
