package jphttpvaultx

import (
	"context"
	"fmt"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// DatabaseCredentials holds the username and password retrieved from Vault for
// a given database service.
type DatabaseCredentials struct {
	Username string
	Password string
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// kvv2DataPath returns the full KV v2 data path for the given secretPath.
// Vault KV v2 puts user data under <mount>/data/<path>.
func (c *Client) kvv2DataPath(secretPath string) string {
	return fmt.Sprintf("%s/data/%s", c.kvMount, secretPath)
}

// readKV fetches a KV v2 secret and unwraps the outer Vault envelope,
// returning only the inner user-written data map.
//
//	GET /v1/<mount>/data/<path>
//	→ { "data": { "data": { … }, "metadata": { … } } }
//	                  ↑ this is what readKV returns
func (c *Client) readKV(ctx context.Context, secretPath string) (map[string]interface{}, error) {
	envelope, err := c.get(ctx, c.kvv2DataPath(secretPath))
	if err != nil {
		return nil, fmt.Errorf("readKV %q: %w", secretPath, err)
	}

	inner, ok := envelope["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("readKV %q: unexpected KV v2 envelope (missing inner data map)", secretPath)
	}

	return inner, nil
}

// stringField extracts a string value from a data map, giving a clear error
// when the key is missing or has the wrong type.
func stringField(data map[string]interface{}, key string) (string, error) {
	raw, ok := data[key]
	if !ok {
		return "", fmt.Errorf("field %q not found in secret", key)
	}
	s, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("field %q is not a string (got %T)", key, raw)
	}
	return s, nil
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// GetSecret fetches a KV v2 secret at secretPath and returns the string value
// for a single key.
//
// Example:
//
//	val, err := client.GetSecret(ctx, "my-service/config", "api_key")
func (c *Client) GetSecret(ctx context.Context, secretPath, key string) (string, error) {
	data, err := c.readKV(ctx, secretPath)
	if err != nil {
		return "", fmt.Errorf("GetSecret: %w", err)
	}
	val, err := stringField(data, key)
	if err != nil {
		return "", fmt.Errorf("GetSecret %q: %w", secretPath, err)
	}
	return val, nil
}

// GetSecretMap fetches a KV v2 secret and returns the entire inner data map.
// Useful when a secret holds multiple keys and you want them all at once.
//
// Example:
//
//	m, err := client.GetSecretMap(ctx, "my-service/config")
//	region := m["region"].(string)
func (c *Client) GetSecretMap(ctx context.Context, secretPath string) (map[string]interface{}, error) {
	data, err := c.readKV(ctx, secretPath)
	if err != nil {
		return nil, fmt.Errorf("GetSecretMap: %w", err)
	}
	return data, nil
}

// GetDatabaseCredentials retrieves database credentials from Vault.
//
// Vault path: <mount>/data/database/<service>
//
// The secret must contain at least the keys "username" and "password".
//
// Example:
//
//	creds, err := client.GetDatabaseCredentials(ctx, "payments-db")
//	if err != nil { log.Fatal(err) }
//	dsn := fmt.Sprintf("postgres://%s:%s@host/dbname",
//	    creds.Username, creds.Password)
func (c *Client) GetDatabaseCredentials(ctx context.Context, service string) (*DatabaseCredentials, error) {
	data, err := c.readKV(ctx, fmt.Sprintf("database/%s", service))
	if err != nil {
		return nil, fmt.Errorf("GetDatabaseCredentials: %w", err)
	}

	username, err := stringField(data, "username")
	if err != nil {
		return nil, fmt.Errorf("GetDatabaseCredentials %q: %w", service, err)
	}

	password, err := stringField(data, "password")
	if err != nil {
		return nil, fmt.Errorf("GetDatabaseCredentials %q: %w", service, err)
	}

	return &DatabaseCredentials{
		Username: username,
		Password: password,
	}, nil
}

// PutSecret writes (or updates) a KV v2 secret at secretPath with the given
// key-value pairs.
//
// Example:
//
//	err := client.PutSecret(ctx, "my-service/config", map[string]any{
//	    "api_key": "abc123",
//	})
func (c *Client) PutSecret(ctx context.Context, secretPath string, values map[string]any) error {
	_, err := c.post(ctx, c.kvv2DataPath(secretPath), map[string]any{
		"data": values,
	})
	if err != nil {
		return fmt.Errorf("PutSecret: failed to write %q: %w", secretPath, err)
	}
	return nil
}

// DeleteSecret permanently deletes a KV v2 secret at secretPath.
func (c *Client) DeleteSecret(ctx context.Context, secretPath string) error {
	if err := c.delete(ctx, c.kvv2DataPath(secretPath)); err != nil {
		return fmt.Errorf("DeleteSecret: failed to delete %q: %w", secretPath, err)
	}
	return nil
}

