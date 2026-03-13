package jphttpvaultx

import (
	"context"
	"fmt"
)

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
