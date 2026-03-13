package jphttpvaultx

import (
	"context"
	"fmt"
)

// GetThirdPartyAppCredential fetches the email and password for a third-party
// application from the Vault path "third-party/<thirdPartyAppSecretPath>".
//
// It delegates to GetSecretMap and then applies unwrapSecrets, so both flat
// and "secrets"-wrapped Vault layouts are accepted:
//
//	Flat:    { "email": "...", "password": "..." }
//	Wrapped: { "secrets": { "email": "...", "password": "..." } }
func (c *Client) GetThirdPartyAppCredential(ctx context.Context, thirdPartyAppSecretPath string) (*ThirdPartyAppCredentials, error) {
	data, err := c.GetSecretMap(ctx, thirdPartyAppSecretPath)
	if err != nil {
		return nil, fmt.Errorf("GetThirdPartyAppCredential: %w", err)
	}

	data = unwrapSecrets(data)

	email, err := stringField(data, "email")
	if err != nil {
		return nil, fmt.Errorf("GetThirdPartyAppCredential %q: %w", thirdPartyAppSecretPath, err)
	}

	password, err := stringField(data, "password")
	if err != nil {
		return nil, fmt.Errorf("GetThirdPartyAppCredential %q: %w", thirdPartyAppSecretPath, err)
	}

	return &ThirdPartyAppCredentials{
		Email:    email,
		Password: password,
	}, nil
}
