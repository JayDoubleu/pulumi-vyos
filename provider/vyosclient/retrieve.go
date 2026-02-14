package vyosclient

import (
	"context"
	"encoding/json"
	"fmt"
)

// ShowConfig retrieves the configuration at the given path.
func (c *Client) ShowConfig(ctx context.Context, path []string) (json.RawMessage, error) {
	anyPath := make([]any, len(path))
	for i, p := range path {
		anyPath[i] = p
	}
	req := map[string]any{
		"op":   "showConfig",
		"path": anyPath,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal showConfig request: %w", err)
	}
	return c.post(ctx, "/retrieve", data)
}

// Exists checks whether a configuration path exists.
func (c *Client) Exists(ctx context.Context, path []string) (bool, error) {
	anyPath := make([]any, len(path))
	for i, p := range path {
		anyPath[i] = p
	}
	req := map[string]any{
		"op":   "exists",
		"path": anyPath,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return false, fmt.Errorf("marshal exists request: %w", err)
	}
	_, err = c.post(ctx, "/retrieve", data)
	if err != nil {
		// VyOS returns success=false when path doesn't exist
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 0 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
