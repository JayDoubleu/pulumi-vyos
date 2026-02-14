// Package vyosclient provides an HTTP client for the VyOS REST API.
package vyosclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
)

// API defines the interface for VyOS HTTP API operations.
// Resources depend on this interface, not the concrete Client, enabling mock injection for tests.
type API interface {
	Set(ctx context.Context, path []string, value any) error
	Delete(ctx context.Context, path []string) error
	BatchConfigure(ctx context.Context, ops []Operation) error
	ShowConfig(ctx context.Context, path []string) (json.RawMessage, error)
	Exists(ctx context.Context, path []string) (bool, error)
	SaveConfig(ctx context.Context) error
}

// Operation represents a single VyOS configure operation.
type Operation struct {
	Op    string `json:"op"`
	Path  []any  `json:"path"`
	Value any    `json:"value,omitempty"`
}

// Client is the concrete VyOS HTTP API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	mu         sync.Mutex
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client (for TLS configuration, timeouts, etc).
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// New creates a new VyOS API client.
func New(baseURL, apiKey string, opts ...Option) *Client {
	c := &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// apiResponse represents the standard VyOS API response envelope.
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *string         `json:"error"`
}

// post sends a POST request to the given VyOS API endpoint.
// It acquires the mutex to ensure only one request is in flight at a time
// (VyOS API is not concurrent-safe).
// The data parameter is JSON-encoded and sent as the "data" form field.
func (c *Client) post(ctx context.Context, endpoint string, data json.RawMessage) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("data", string(data)); err != nil {
		return nil, fmt.Errorf("write data field: %w", err)
	}
	if err := w.WriteField("key", c.apiKey); err != nil {
		return nil, fmt.Errorf("write key field: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request to %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	const maxResponseSize = 10 << 20 // 10 MiB
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if len(respBody) > maxResponseSize {
		return nil, fmt.Errorf("response body exceeds %d bytes", maxResponseSize)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &AuthError{Message: string(respBody)}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(respBody)}
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !apiResp.Success {
		msg := "unknown error"
		if apiResp.Error != nil {
			msg = *apiResp.Error
		}
		return nil, &APIError{StatusCode: 0, Message: msg}
	}

	return apiResp.Data, nil
}

// Compile-time check that Client implements API.
var _ API = (*Client)(nil)
