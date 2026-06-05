package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth"
)

type oauthAuthenticator struct {
	newTransport TransportFactory
}

func NewOAuthAuthenticator(oauthSessionStore oauth.Store, oauthCallback oauth.Callback) Authenticator {
	return &oauthAuthenticator{
		newTransport: NewTransportFactory(oauthSessionStore, oauthCallback),
	}
}

func (a *oauthAuthenticator) Authenticate(ctx context.Context, cfg Config) error {
	if cfg.OAuth == nil || !cfg.OAuth.IsEnabled() {
		return fmt.Errorf("mcp: client %q has no oauth configuration", cfg.Name)
	}

	if !cfg.IsEnabled() {
		return fmt.Errorf("mcp: client %q is disabled", cfg.Name)
	}

	ctx, cancel := withTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	transport, err := a.newTransport(cfg)
	if err != nil {
		return err
	}

	sdk := mcpsdk.NewClient(
		&mcpsdk.Implementation{
			Name:    clientName,
			Version: clientVersion,
		},
		nil,
	)

	session, err := sdk.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("mcp: authenticate: %w", err)
	}

	return session.Close()
}
