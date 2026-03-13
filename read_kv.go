package jphttpvaultx

import (
	"context"
	"fmt"
)

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
