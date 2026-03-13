package jphttpvaultx

import (
	"context"
	"fmt"
)

// GetSecretKey fetches the value of the "secret_key" field at the given path.
//
// It delegates to GetSecretMap and then applies unwrapSecrets, so both flat
// and "secrets"-wrapped Vault layouts are accepted:
//
//	Flat:    { "secret_key": "..." }
//	Wrapped: { "secrets": { "secret_key": "..." } }
func (c *Client) GetSecretKey(ctx context.Context, secretKeyPath string) (string, error) {
	data, err := c.GetSecret(ctx, secretKeyPath, "secret_key")
	if err != nil {
		return "", fmt.Errorf("GetSecretKey: %w", err)
	}

	return data, nil
}
