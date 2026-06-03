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

	HTTPClient   *http.Client
	Callback     Callback
	Registration RegistrationInfo
}

type handler struct {
	config      HandlerConfig
	interactive bool

	mu     sync.Mutex
	source oauth2.TokenSource
}

func NewPassiveHandler(config HandlerConfig) mcpauth.OAuthHandler {
	return &handler{config: config, interactive: false}
}

func NewInteractiveHandler(config HandlerConfig) mcpauth.OAuthHandler {
	return &handler{config: config, interactive: true}
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

	h.source = h.refreshingSource(*session)

	return h.source, nil
}

func (h *handler) Authorize(ctx context.Context, _ *http.Request, resp *http.Response) error {
	if !h.interactive {
		closeResponse(resp)

		return ErrAuthorizationRequired
	}

	session, err := h.authorize(ctx, resp)
	if err != nil {
		return err
	}

	if err := h.config.Store.Save(ctx, h.config.Key, session); err != nil {
		return err
	}

	h.mu.Lock()
	h.source = h.refreshingSource(session)
	h.mu.Unlock()

	return nil
}

func (h *handler) refreshingSource(session Session) oauth2.TokenSource {
	cfg := session.oauthConfig(h.config.RedirectURL)
	base := cfg.TokenSource(h.clientContext(context.Background()), session.oauthToken())

	return &persistingSource{
		base:    base,
		key:     h.config.Key,
		store:   h.config.Store,
		session: session,
	}
}

func (h *handler) clientContext(ctx context.Context) context.Context {
	if h.config.HTTPClient == nil {
		return ctx
	}

	return context.WithValue(ctx, oauth2.HTTPClient, h.config.HTTPClient)
}
