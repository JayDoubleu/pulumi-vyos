//go:build integration

package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
)

// vyosClient creates a VyOS API client configured for the test VM.
// Connection details come from environment variables with sensible defaults
// matching the test VM (test/vm/run.sh).
func vyosClient(t *testing.T) *vyosclient.Client {
	t.Helper()

	host := envOrDefault("VYOS_HOST", "localhost")
	port := envOrDefault("VYOS_API_PORT", "8443")
	apiKey := envOrDefault("VYOS_API_KEY", "integration-test-key")

	baseURL := fmt.Sprintf("https://%s:%s", host, port)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test VM uses self-signed cert
		},
	}

	client := vyosclient.New(baseURL, apiKey, vyosclient.WithHTTPClient(httpClient))
	skipIfVMUnavailable(t, client)
	return client
}

// skipIfVMUnavailable skips the test if the VyOS VM is not reachable.
func skipIfVMUnavailable(t *testing.T, client *vyosclient.Client) {
	t.Helper()

	_, err := client.ShowConfig(context.Background(), []string{"system"})
	if err != nil {
		t.Skipf("VyOS VM not reachable: %v", err)
	}
}

// vyosClientWithKey creates a VyOS API client with a custom API key.
// Unlike vyosClient, it does not call skipIfVMUnavailable, so callers
// can test authentication failures against a running VM.
func vyosClientWithKey(t *testing.T, apiKey string) *vyosclient.Client {
	t.Helper()

	host := envOrDefault("VYOS_HOST", "localhost")
	port := envOrDefault("VYOS_API_PORT", "8443")

	baseURL := fmt.Sprintf("https://%s:%s", host, port)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test VM uses self-signed cert
		},
	}

	return vyosclient.New(baseURL, apiKey, vyosclient.WithHTTPClient(httpClient))
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// deleteIfExists deletes the VyOS config at path, ignoring errors if it
// doesn't exist. Useful in t.Cleanup to ensure a clean state.
func deleteIfExists(t *testing.T, client *vyosclient.Client, path []string) {
	t.Helper()
	_ = client.Delete(context.Background(), path)
}
