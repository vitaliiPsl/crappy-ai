package mcp

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth"
	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
)

type TransportFactory func(Config) (mcpsdk.Transport, error)

func NewTransportFactory(oauthSessionStore oauth.Store, oauthCallback appoauth.Callback) TransportFactory {
	return func(cfg Config) (mcpsdk.Transport, error) {
		return newTransport(cfg, oauthSessionStore, oauthCallback)
	}
}

func newTransport(cfg Config, oauthSessionStore oauth.Store, oauthCallback appoauth.Callback) (mcpsdk.Transport, error) {
	switch cfg.Transport {
	case "", TransportStdio:
		return newStdioTransport(cfg)
	case TransportHTTP:
		return newHTTPTransport(cfg, oauthSessionStore, oauthCallback)
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

func newHTTPTransport(cfg Config, oauthSessionStore oauth.Store, oauthCallback appoauth.Callback) (mcpsdk.Transport, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("mcp: client %q has no url for http transport", cfg.Name)
	}

	httpClient, err := httpClientWithStaticHeaders(cfg)
	if err != nil {
		return nil, err
	}

	oauthHandler, err := newOAuthHandler(cfg, httpClient, oauthSessionStore, oauthCallback)
	if err != nil {
		return nil, fmt.Errorf("mcp: client %q oauth: %w", cfg.Name, err)
	}

	return &mcpsdk.StreamableClientTransport{
		Endpoint:     cfg.URL,
		HTTPClient:   httpClient,
		OAuthHandler: oauthHandler,
	}, nil
}

func newOAuthHandler(cfg Config, httpClient *http.Client, oauthSessionStore oauth.Store, oauthCallback appoauth.Callback) (mcpauth.OAuthHandler, error) {
	if cfg.OAuth == nil || !cfg.OAuth.IsEnabled() {
		return nil, nil
	}

	redirectURL, err := oauth.RedirectURL(*cfg.OAuth)
	if err != nil {
		return nil, err
	}

	config := oauth.HandlerConfig{
		Key:         oauth.NewKey(cfg.Name, cfg.URL),
		Store:       oauthSessionStore,
		RedirectURL: redirectURL,
		Scopes:      cfg.OAuth.Scopes,
		HTTPClient:  httpClient,
		Registration: oauth.RegistrationInfo{
			ClientID:     cfg.OAuth.ClientID,
			ClientSecret: cfg.OAuth.ResolveClientSecret(),
			ClientName:   clientName,
			SoftwareID:   clientName,
			Version:      clientVersion,
		},
	}

	if oauthCallback != nil {
		config.Callback = oauthCallback
	}

	return oauth.New(config), nil
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
