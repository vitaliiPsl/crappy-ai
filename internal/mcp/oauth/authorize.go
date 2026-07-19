package oauth

import (
	"context"
	"errors"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"

	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
)

type AuthorizerConfig struct {
	Key          Key
	RedirectURL  string
	Scopes       []string
	Callback     appoauth.Callback
	HTTPClient   *http.Client
	Registration RegistrationInfo
}

type Authorizer struct {
	config AuthorizerConfig
}

func NewAuthorizer(config AuthorizerConfig) *Authorizer {
	return &Authorizer{config: config}
}

func (a *Authorizer) Authorize(ctx context.Context, resp *http.Response) (Session, error) {
	discovery, err := Discover(ctx, a.config.HTTPClient, a.config.Key.ServerURL, resp)
	if err != nil {
		return Session{}, err
	}

	clientID, clientSecret, err := a.client(ctx, discovery.RegistrationURL)
	if err != nil {
		return Session{}, err
	}

	session := Session{
		ServerURL:    a.config.Key.ServerURL,
		Resource:     discovery.Resource,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      discovery.AuthorizationURL,
		TokenURL:     discovery.TokenURL,
		Scopes:       a.scopes(discovery.Scopes),
	}

	token, err := a.exchange(ctx, session.oauthConfig(a.config.RedirectURL), discovery.Resource)
	if err != nil {
		return Session{}, err
	}

	return withToken(session, token), nil
}

func (a *Authorizer) client(ctx context.Context, registrationURL string) (string, string, error) {
	if a.config.Registration.ClientID != "" {
		return a.config.Registration.ClientID, a.config.Registration.ClientSecret, nil
	}

	if registrationURL == "" {
		return "", "", errors.New("oauth: server has no registration endpoint and no client_id is configured")
	}

	resp, err := oauthex.RegisterClient(ctx, registrationURL, &oauthex.ClientRegistrationMetadata{
		RedirectURIs:            []string{a.config.RedirectURL},
		TokenEndpointAuthMethod: "none",
		ClientName:              a.config.Registration.ClientName,
		SoftwareID:              a.config.Registration.SoftwareID,
		SoftwareVersion:         a.config.Registration.Version,
	}, a.config.HTTPClient)
	if err != nil {
		return "", "", err
	}

	return resp.ClientID, resp.ClientSecret, nil
}

func (a *Authorizer) exchange(ctx context.Context, cfg oauth2.Config, resource string) (*oauth2.Token, error) {
	resourceParam := oauth2.SetAuthURLParam("resource", resource)
	flow := appoauth.CodeFlow{Config: cfg, HTTPClient: a.config.HTTPClient}

	return flow.Authorize(ctx, a.config.Callback, appoauth.CodeFlowOptions{
		Authorization: []oauth2.AuthCodeOption{oauth2.AccessTypeOffline, resourceParam},
		Token:         []oauth2.AuthCodeOption{resourceParam},
	})
}

func (a *Authorizer) scopes(discovered []string) []string {
	if len(a.config.Scopes) > 0 {
		return a.config.Scopes
	}

	return discovered
}
