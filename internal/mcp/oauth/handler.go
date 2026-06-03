package oauth

import (
	"context"
	"errors"
	"net/http"
	"sync"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"golang.org/x/oauth2"
)

var ErrAuthorizationRequired = errors.New("mcp: oauth authorization required")

type Callback interface {
	Wait(ctx context.Context, authURL string) (code string, state string, err error)
}

type RegistrationInfo struct {
	ClientID     string
	ClientSecret string
	ClientName   string
	SoftwareID   string
	Version      string
}

type HandlerConfig struct {
	Key         Key
	Store       Store
	RedirectURL string
	Scopes      []string

	Callback     Callback
	HTTPClient   *http.Client
	Registration RegistrationInfo
}

type handler struct {
	config     HandlerConfig
	authorizer *Authorizer

	mu     sync.Mutex
	source oauth2.TokenSource
}

func New(config HandlerConfig) mcpauth.OAuthHandler {
	authorizer := NewAuthorizer(AuthorizerConfig{
		Key:          config.Key,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
		Callback:     config.Callback,
		HTTPClient:   config.HTTPClient,
		Registration: config.Registration,
	})

	return &handler{
		config:     config,
		authorizer: authorizer,
	}
}

func (h *handler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.source != nil {
		return h.source, nil
	}

	session, err := h.config.Store.Load(ctx, h.config.Key)
	if err != nil || session == nil || !session.hasToken() {
		return nil, err
	}

	h.source = newPersistingSource(*session, h.config.Key, h.config.Store)

	return h.source, nil
}

func (h *handler) Authorize(ctx context.Context, _ *http.Request, resp *http.Response) error {
	if h.config.Callback == nil {
		closeResponse(resp)

		return ErrAuthorizationRequired
	}

	session, err := h.authorizer.Authorize(ctx, resp)
	if err != nil {
		return err
	}

	if err := h.config.Store.Save(ctx, h.config.Key, session); err != nil {
		return err
	}

	h.mu.Lock()
	h.source = newPersistingSource(session, h.config.Key, h.config.Store)
	h.mu.Unlock()

	return nil
}
