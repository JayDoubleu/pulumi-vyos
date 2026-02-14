package vyosclient

import (
	"context"
	"encoding/json"
	"fmt"
)

// SaveConfig persists the running configuration to disk.
func (c *Client) SaveConfig(ctx context.Context) error {
	req := map[string]any{
		"op": "save",
	}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal save request: %w", err)
	}
	_, err = c.post(ctx, "/config-file", data)
	return err
}
