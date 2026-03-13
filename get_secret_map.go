package jphttpvaultx

import (
	"context"
	"fmt"
)

func (c *Client) GetSecretMap(ctx context.Context, secretPath string) (map[string]interface{}, error) {
	data, err := c.readKV(ctx, secretPath)
	if err != nil {
		return nil, fmt.Errorf("GetSecretMap: %w", err)
	}
	return data, nil
}
