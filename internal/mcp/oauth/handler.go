package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
)

// ErrAuthorizationRequired is returned by the passive handler when a server asks for OAuth.
var ErrAuthorizationRequired = errors.New("mcp: oauth authorization required")

type HandlerConfig struct {
	Config      *Config
	ClientName  string
	ClientLabel string
	Version     string
	HTTPClient  *http.Client
	Interactive bool
}

func NewHandler(config HandlerConfig) (mcpauth.OAuthHandler, error) {
	if config.Config == nil || !config.Config.IsEnabled() {
		return nil, nil
	}

	if !config.Interactive {
		return passiveHandler{}, nil
	}

	return newInteractiveHandler(config)
}

type passiveHandler struct{}

func (passiveHandler) TokenSource(context.Context) (oauth2.TokenSource, error) {
	return nil, nil
}

func (passiveHandler) Authorize(_ context.Context, _ *http.Request, resp *http.Response) error {
	defer func() { _ = resp.Body.Close() }()

	return ErrAuthorizationRequired
}

type interactiveHandler struct {
	delegate mcpauth.OAuthHandler
}

func newInteractiveHandler(config HandlerConfig) (*interactiveHandler, error) {
	redirectURL, err := RedirectURL(*config.Config)
	if err != nil {
		return nil, err
	}

	handlerConfig, err := authorizationCodeConfig(config, redirectURL)
	if err != nil {
		return nil, err
	}

	handler, err := mcpauth.NewAuthorizationCodeHandler(handlerConfig)
	if err != nil {
		return nil, fmt.Errorf("configure authorization code handler: %w", err)
	}

	return &interactiveHandler{
		delegate: handler,
	}, nil
}

func (h *interactiveHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	return h.delegate.TokenSource(ctx)
}

func (h *interactiveHandler) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	return h.delegate.Authorize(ctx, req, resp)
}

func authorizationCodeConfig(config HandlerConfig, redirectURL string) (*mcpauth.AuthorizationCodeHandlerConfig, error) {
	server := NewCallbackServer(
		redirectURL,
		NewBrowserPrompter(),
	)

	cfg := &mcpauth.AuthorizationCodeHandlerConfig{
		Client:                   config.HTTPClient,
		RedirectURL:              redirectURL,
		AuthorizationCodeFetcher: server.Fetch,
	}

	configureClientIDMetadata(cfg, *config.Config)

	configurePreregisteredClient(cfg, *config.Config)

	configureDynamicRegistration(cfg, config, redirectURL)

	return cfg, nil
}

func configureClientIDMetadata(cfg *mcpauth.AuthorizationCodeHandlerConfig, config Config) {
	if config.ClientIDMetadataURL == "" {
		return
	}

	cfg.ClientIDMetadataDocumentConfig = &mcpauth.ClientIDMetadataDocumentConfig{
		URL: config.ClientIDMetadataURL,
	}
}

func configurePreregisteredClient(cfg *mcpauth.AuthorizationCodeHandlerConfig, config Config) {
	if config.ClientID == "" {
		return
	}

	client := clientCredentials(config)
	cfg.PreregisteredClient = client
}

func configureDynamicRegistration(cfg *mcpauth.AuthorizationCodeHandlerConfig, config HandlerConfig, redirectURL string) {
	if !config.Config.UsesDynamicRegistration() {
		return
	}

	cfg.DynamicClientRegistrationConfig = &mcpauth.DynamicClientRegistrationConfig{
		Metadata: &oauthex.ClientRegistrationMetadata{
			RedirectURIs:            []string{redirectURL},
			TokenEndpointAuthMethod: "none",
			ClientName:              config.ClientLabel,
			SoftwareID:              config.ClientName,
			SoftwareVersion:         config.Version,
		},
	}
}

func clientCredentials(cfg Config) *oauthex.ClientCredentials {
	secret := cfg.ClientSecret
	if cfg.ClientSecretEnv != "" {
		secret = os.Getenv(cfg.ClientSecretEnv)
	}

	client := &oauthex.ClientCredentials{ClientID: cfg.ClientID}
	if secret != "" {
		client.ClientSecretAuth = &oauthex.ClientSecretAuth{ClientSecret: secret}
	}

	return client
}
