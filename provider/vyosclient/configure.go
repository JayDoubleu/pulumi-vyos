package vyosclient

import (
	"context"
	"encoding/json"
	"fmt"
)

// Set applies a single set operation on the VyOS configuration.
func (c *Client) Set(ctx context.Context, path []string, value any) error {
	anyPath := make([]any, len(path))
	for i, p := range path {
		anyPath[i] = p
	}
	op := Operation{Op: "set", Path: anyPath, Value: value}
	return c.configure(ctx, op)
}

// Delete removes a configuration path from VyOS.
func (c *Client) Delete(ctx context.Context, path []string) error {
	anyPath := make([]any, len(path))
	for i, p := range path {
		anyPath[i] = p
	}
	op := Operation{Op: "delete", Path: anyPath}
	return c.configure(ctx, op)
}

// BatchConfigure sends multiple operations in a single atomic commit.
func (c *Client) BatchConfigure(ctx context.Context, ops []Operation) error {
	return c.configureBatch(ctx, ops)
}

// configure sends a single operation to the /configure endpoint.
func (c *Client) configure(ctx context.Context, op Operation) error {
	data, err := json.Marshal(op)
	if err != nil {
		return fmt.Errorf("marshal configure request: %w", err)
	}
	_, err = c.post(ctx, "/configure", data)
	return err
}

// configureBatch sends an array of operations to the /configure endpoint.
func (c *Client) configureBatch(ctx context.Context, ops []Operation) error {
	data, err := json.Marshal(ops)
	if err != nil {
		return fmt.Errorf("marshal batch configure request: %w", err)
	}
	_, err = c.post(ctx, "/configure", data)
	return err
}
