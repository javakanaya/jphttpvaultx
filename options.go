package jphttpvaultx

import (
	"net/http"
	"time"
)

// Option is a functional option for configuring a Client.
type Option func(*Client)

// WithProxyAddr overrides the Vault Lambda Extension proxy address.
// Defaults to http://127.0.0.1:8200 (the extension's default).
func WithProxyAddr(addr string) Option {
	return func(c *Client) {
		c.proxyAddr = addr
	}
}

// WithNamespace sets the Vault Enterprise namespace header (X-Vault-Namespace).
// Pass an empty string (or omit this option) for Vault OSS / no namespace.
func WithNamespace(ns string) Option {
	return func(c *Client) {
		c.namespace = ns
	}
}

// WithTimeout sets a custom HTTP timeout.
// Defaults to 5 seconds if not specified.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithHTTPClient replaces the underlying *http.Client entirely.
// Useful when you need custom TLS config, transport settings, etc.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithKVMount overrides the KV secrets engine mount path.
// Defaults to "secret" (the default KV v2 mount in Vault dev mode).
// For example, use "static-secret" if that is your mount name.
func WithKVMount(mount string) Option {
	return func(c *Client) {
		c.kvMount = mount
	}
}
