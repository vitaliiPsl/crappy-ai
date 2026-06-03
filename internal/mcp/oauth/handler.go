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

	HTTPClient *http.Client
	Prompter   Prompter

	SessionKey   SessionKey
	SessionStore SessionStore
}

type handler struct {
	authorizer    mcpauth.OAuthHandler
	authorization authorizationBehavior
	sessionKey    SessionKey
	sessionStore  SessionStore
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
		sessionKey:    config.SessionKey,
		sessionStore:  config.SessionStore,
	}, nil
}

func (h *handler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	source, err := h.authorizer.TokenSource(ctx)
	if err != nil {
		return nil, err
	}

	if source != nil {
		return h.savingTokenSource(source), nil
	}

	return h.storedTokenSource(ctx)
}

func (h *handler) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	if err := h.authorization.Authorize(ctx, h.authorizer, req, resp); err != nil {
		return err
	}

	return h.saveAuthorizerToken(ctx)
}

func (h *handler) storedTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	if h.sessionStore == nil {
		return nil, nil
	}

	session, err := h.sessionStore.Load(ctx, h.sessionKey)
	if err != nil {
		return nil, err
	}

	if session == nil {
		return nil, nil
	}

	token := session.oauthToken()
	if !token.Valid() {
		return nil, nil
	}

	return oauth2.StaticTokenSource(token), nil
}

func (h *handler) saveAuthorizerToken(ctx context.Context) error {
	if h.sessionStore == nil {
		return nil
	}

	source, err := h.authorizer.TokenSource(ctx)
	if err != nil || source == nil {
		return err
	}

	token, err := source.Token()
	if err != nil || token == nil || token.AccessToken == "" {
		return err
	}

	return h.sessionStore.Save(ctx, h.sessionKey, sessionFromToken(h.sessionKey.ServerURL, token))
}

func (h *handler) savingTokenSource(source oauth2.TokenSource) oauth2.TokenSource {
	if h.sessionStore == nil {
		return source
	}

	return newSavingTokenSource(source, h.sessionKey, h.sessionStore)
}
