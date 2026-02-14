package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Config defines the provider configuration for connecting to a VyOS instance.
type Config struct {
	Host       string  `pulumi:"host"`
	APIKey     string  `pulumi:"apiKey" provider:"secret"`
	Port       *int    `pulumi:"port,optional"`
	Protocol   *string `pulumi:"protocol,optional"`
	Insecure   *bool   `pulumi:"insecure,optional"`
	SaveConfig *bool   `pulumi:"saveConfig,optional"`

	client vyosclient.API
}

// Annotate provides schema metadata for the provider configuration.
func (c *Config) Annotate(a infer.Annotator) {
	a.Describe(&c.Host, "The VyOS host address (IP or hostname).")
	a.Describe(&c.APIKey, "The VyOS HTTP API key.")
	a.Describe(&c.Port, "The API port. Defaults to 443.")
	a.Describe(&c.Protocol, "The protocol to use (https or http). Defaults to https.")
	a.Describe(&c.Insecure, "Skip TLS certificate verification. Defaults to false.")
	a.Describe(&c.SaveConfig, "Automatically save running config to disk after every mutating operation. Defaults to false.")
	a.SetDefault(&c.Port, 443)
	a.SetDefault(&c.Protocol, "https")
	a.SetDefault(&c.Insecure, false)
	a.SetDefault(&c.SaveConfig, false)
}

// Configure initializes the VyOS API client from provider configuration.
func (c *Config) Configure(_ context.Context) error {
	protocol := "https"
	if c.Protocol != nil {
		protocol = *c.Protocol
	}
	port := 443
	if c.Port != nil {
		port = *c.Port
	}
	insecure := false
	if c.Insecure != nil {
		insecure = *c.Insecure
	}

	baseURL := fmt.Sprintf("%s://%s", protocol, net.JoinHostPort(c.Host, strconv.Itoa(port)))

	httpClient := &http.Client{
		Timeout: 120 * time.Second,
	}
	if insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // User-requested TLS skip for self-signed certs.
			},
		}
	}

	c.client = vyosclient.New(baseURL, c.APIKey, vyosclient.WithHTTPClient(httpClient))
	return nil
}

// Client returns the configured VyOS API client.
func (c Config) Client() vyosclient.API {
	return c.client
}

// getClient retrieves the VyOS API client from the provider config context.
func getClient(ctx context.Context) vyosclient.API {
	return infer.GetConfig[Config](ctx).Client()
}

// ShouldSaveConfig reports whether the provider is configured to auto-save
// the running config after mutating operations.
func (c Config) ShouldSaveConfig() bool {
	return c.SaveConfig != nil && *c.SaveConfig
}

// saveIfEnabled saves the running config to disk when the provider's
// saveConfig flag is set. Failures are logged as warnings rather than
// returned as errors because the CRUD operation itself already succeeded.
func saveIfEnabled(ctx context.Context) {
	cfg := infer.GetConfig[Config](ctx)
	if !cfg.ShouldSaveConfig() {
		return
	}
	if err := cfg.Client().SaveConfig(ctx); err != nil {
		p.GetLogger(ctx).Warningf("auto-save config failed: %v", err)
	}
}
