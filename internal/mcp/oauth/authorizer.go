package oauth

import (
	"fmt"
	"os"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

func newAuthorizer(config HandlerConfig) (mcpauth.OAuthHandler, error) {
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

	return handler, nil
}

func authorizationCodeConfig(config HandlerConfig, redirectURL string) (*mcpauth.AuthorizationCodeHandlerConfig, error) {
	server := NewCallbackServer(
		redirectURL,
		config.Prompter,
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
