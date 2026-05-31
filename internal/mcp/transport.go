package mcp

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *sdkClient) transport() (mcpsdk.Transport, error) {
	switch c.config.Transport {
	case "", TransportStdio:
		return c.stdioTransport()
	case TransportHTTP:
		return c.httpTransport()
	default:
		return nil, fmt.Errorf("mcp: client %q has unsupported transport %q", c.config.Name, c.config.Transport)
	}
}

func (c *sdkClient) stdioTransport() (mcpsdk.Transport, error) {
	if c.config.Command == "" {
		return nil, fmt.Errorf("mcp: client %q has no command for stdio transport", c.config.Name)
	}

	cmd := exec.CommandContext(c.ctx, c.config.Command, c.config.Args...)
	cmd.Env = append(os.Environ(), c.config.Env...)

	return &mcpsdk.CommandTransport{
		Command: cmd,
	}, nil
}

func (c *sdkClient) httpTransport() (mcpsdk.Transport, error) {
	if c.config.URL == "" {
		return nil, fmt.Errorf("mcp: client %q has no url for http transport", c.config.Name)
	}

	httpClient, err := c.httpClient()
	if err != nil {
		return nil, err
	}

	return &mcpsdk.StreamableClientTransport{
		Endpoint:   c.config.URL,
		HTTPClient: httpClient,
	}, nil
}

func (c *sdkClient) httpClient() (*http.Client, error) {
	headers, err := c.authHeaders()
	if err != nil {
		return nil, err
	}

	if len(headers) == 0 {
		return nil, nil
	}

	return &http.Client{
		Transport: authTransport{
			base:    http.DefaultTransport,
			headers: headers,
		},
	}, nil
}

func (c *sdkClient) authHeaders() (http.Header, error) {
	headers := make(http.Header)

	for name, value := range c.config.Auth.Headers {
		headers.Set(name, value)
	}

	for name, env := range c.config.Auth.HeaderEnv {
		value := os.Getenv(env)
		if value == "" {
			return nil, fmt.Errorf("mcp: client %q auth header %q references empty env %q", c.config.Name, name, env)
		}

		headers.Set(name, value)
	}

	return headers, nil
}

type authTransport struct {
	base    http.RoundTripper
	headers http.Header
}

func (t authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header = req.Header.Clone()

	for name, values := range t.headers {
		for _, value := range values {
			cloned.Header.Set(name, value)
		}
	}

	return t.base.RoundTrip(cloned)
}
