package jphttpvaultx

import (
	"context"
	"fmt"
)

// GetDatabaseCredentials fetches the username and password for a database from
// the Vault path "database/<databaseSecretPath>".
//
// It delegates to GetSecretMap and then applies unwrapSecrets, so both flat
// and "secrets"-wrapped Vault layouts are accepted:
//
//	Flat:    { "username": "...", "password": "..." }
//	Wrapped: { "secrets": { "username": "...", "password": "..." } }
func (c *Client) GetDatabaseCredentials(ctx context.Context, databaseSecretPath string) (*DatabaseCredentials, error) {
	data, err := c.GetSecretMap(ctx, databaseSecretPath)
	if err != nil {
		return nil, fmt.Errorf("GetDatabaseCredentials: %w", err)
	}

	data = unwrapSecrets(data)

	username, err := stringField(data, "username")
	if err != nil {
		return nil, fmt.Errorf("GetDatabaseCredentials %q: %w", databaseSecretPath, err)
	}

	password, err := stringField(data, "password")
	if err != nil {
		return nil, fmt.Errorf("GetDatabaseCredentials %q: %w", databaseSecretPath, err)
	}

	return &DatabaseCredentials{
		Username: username,
		Password: password,
	}, nil
}
