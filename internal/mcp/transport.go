package mcp

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth"
)

type transportOptions struct {
	OAuthInteractive bool
	OAuthPrompter    oauth.Prompter
}

func newTransport(cfg Config, opts transportOptions) (mcpsdk.Transport, error) {
	switch cfg.Transport {
	case "", TransportStdio:
		return newStdioTransport(cfg)
	case TransportHTTP:
		return newHTTPTransport(cfg, opts)
	default:
		return nil, fmt.Errorf("mcp: client %q has unsupported transport %q", cfg.Name, cfg.Transport)
	}
}

func newStdioTransport(cfg Config) (mcpsdk.Transport, error) {
	if cfg.Command == "" {
		return nil, fmt.Errorf("mcp: client %q has no command for stdio transport", cfg.Name)
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Env = append(os.Environ(), cfg.Env...)

	return &mcpsdk.CommandTransport{
		Command: cmd,
	}, nil
}

func newHTTPTransport(cfg Config, opts transportOptions) (mcpsdk.Transport, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("mcp: client %q has no url for http transport", cfg.Name)
	}

	httpClient, err := httpClientWithStaticHeaders(cfg)
	if err != nil {
		return nil, err
	}

	handlerConfig := oauth.HandlerConfig{
		Config:      cfg.OAuth,
		ClientName:  clientName,
		ClientLabel: clientName,
		Version:     clientVersion,
		HTTPClient:  httpClient,
		Prompter:    opts.OAuthPrompter,
	}

	oauthHandler, err := newOAuthHandler(handlerConfig, opts)
	if err != nil {
		return nil, fmt.Errorf("mcp: client %q oauth: %w", cfg.Name, err)
	}

	return &mcpsdk.StreamableClientTransport{
		Endpoint:     cfg.URL,
		HTTPClient:   httpClient,
		OAuthHandler: oauthHandler,
	}, nil
}

func newOAuthHandler(config oauth.HandlerConfig, opts transportOptions) (mcpauth.OAuthHandler, error) {
	if opts.OAuthInteractive {
		return oauth.NewInteractiveHandler(config)
	}

	return oauth.NewPassiveHandler(config)
}

func httpClientWithStaticHeaders(cfg Config) (*http.Client, error) {
	headers, err := staticHeaders(cfg)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: staticHeaderTransport{
			base:    http.DefaultTransport,
			headers: headers,
		},
	}, nil
}

func staticHeaders(cfg Config) (http.Header, error) {
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

type staticHeaderTransport struct {
	base    http.RoundTripper
	headers http.Header
}

func (t staticHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header = req.Header.Clone()

	for name, values := range t.headers {
		if cloned.Header.Get(name) != "" {
			continue
		}

		for _, value := range values {
			cloned.Header.Set(name, value)
		}
	}

	return t.base.RoundTrip(cloned)
}
