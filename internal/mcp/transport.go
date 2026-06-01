package mcp

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func buildTransport(cfg Config) (mcpsdk.Transport, error) {
	switch cfg.Transport {
	case "", TransportStdio:
		return stdioTransport(cfg)
	case TransportHTTP:
		return httpTransport(cfg)
	default:
		return nil, fmt.Errorf("mcp: client %q has unsupported transport %q", cfg.Name, cfg.Transport)
	}
}

func stdioTransport(cfg Config) (mcpsdk.Transport, error) {
	if cfg.Command == "" {
		return nil, fmt.Errorf("mcp: client %q has no command for stdio transport", cfg.Name)
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Env = append(os.Environ(), cfg.Env...)

	return &mcpsdk.CommandTransport{
		Command: cmd,
	}, nil
}

func httpTransport(cfg Config) (mcpsdk.Transport, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("mcp: client %q has no url for http transport", cfg.Name)
	}

	httpClient, err := authHTTPClient(cfg)
	if err != nil {
		return nil, err
	}

	return &mcpsdk.StreamableClientTransport{
		Endpoint:   cfg.URL,
		HTTPClient: httpClient,
	}, nil
}

func authHTTPClient(cfg Config) (*http.Client, error) {
	headers, err := headers(cfg)
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

func headers(cfg Config) (http.Header, error) {
	headers := make(http.Header)

	for name, value := range cfg.Headers {
		headers.Set(name, value)
	}

	for name, env := range cfg.HeaderEnv {
		value := os.Getenv(env)
		if value == "" {
			return nil, fmt.Errorf("mcp: client %q auth header %q references empty env %q", cfg.Name, name, env)
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
