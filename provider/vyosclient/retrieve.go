package vyosclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
// VyOS returns {success: true, data: true/false} to indicate presence.
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
	respData, err := c.post(ctx, "/retrieve", data)
	if err != nil {
		// Some VyOS versions return success=false when path doesn't exist.
		// Only treat known "not found" messages as false; other StatusCode==0
		// errors (internal failures, malformed paths) should propagate.
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 0 &&
			(strings.Contains(apiErr.Message, "specified path") ||
				strings.Contains(apiErr.Message, "does not exist")) {
			return false, nil
		}
		return false, err
	}
	var exists bool
	if err := json.Unmarshal(respData, &exists); err != nil {
		return false, fmt.Errorf("unmarshal exists response: %w", err)
	}
	return exists, nil
}
