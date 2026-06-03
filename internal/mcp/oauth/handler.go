package oauth

import (
	"context"
	"errors"
	"net/http"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"golang.org/x/oauth2"
)

// ErrAuthorizationRequired is returned when authorization requires user interaction.
var ErrAuthorizationRequired = errors.New("mcp: oauth authorization required")

type HandlerConfig struct {
	Config      *Config
	ClientName  string
	ClientLabel string
	Version     string
	HTTPClient  *http.Client
	Prompter    Prompter
}

type handler struct {
	authorizer    mcpauth.OAuthHandler
	authorization authorizationBehavior
}

func NewPassiveHandler(config HandlerConfig) (mcpauth.OAuthHandler, error) {
	return newHandler(config, passiveAuthorization{})
}

func NewInteractiveHandler(config HandlerConfig) (mcpauth.OAuthHandler, error) {
	return newHandler(config, interactiveAuthorization{})
}

func newHandler(config HandlerConfig, authorization authorizationBehavior) (mcpauth.OAuthHandler, error) {
	if config.Config == nil || !config.Config.IsEnabled() {
		return nil, nil
	}

	authorizer, err := newAuthorizer(config)
	if err != nil {
		return nil, err
	}

	return &handler{
		authorizer:    authorizer,
		authorization: authorization,
	}, nil
}

func (h *handler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	return h.authorizer.TokenSource(ctx)
}

func (h *handler) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	return h.authorization.Authorize(ctx, h.authorizer, req, resp)
}
