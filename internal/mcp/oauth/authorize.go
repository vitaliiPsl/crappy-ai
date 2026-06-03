package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
)

type endpoints struct {
	resource        string
	authURL         string
	tokenURL        string
	registrationURL string
	scopes          []string
}

type AuthorizerConfig struct {
	Key          Key
	RedirectURL  string
	Scopes       []string
	Callback     Callback
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
	challenge := readChallenge(resp)

	found, err := a.discover(ctx, challenge)
	if err != nil {
		return Session{}, err
	}

	clientID, clientSecret, err := a.client(ctx, found.registrationURL)
	if err != nil {
		return Session{}, err
	}

	session := Session{
		ServerURL:    a.config.Key.ServerURL,
		Resource:     found.resource,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      found.authURL,
		TokenURL:     found.tokenURL,
		Scopes:       a.scopes(found.scopes),
	}

	token, err := a.exchange(ctx, session.oauthConfig(a.config.RedirectURL), found.resource)
	if err != nil {
		return Session{}, err
	}

	return withToken(session, token), nil
}

func (a *Authorizer) discover(ctx context.Context, metadataURL string) (endpoints, error) {
	prm, err := a.resourceMetadata(ctx, metadataURL)
	if err != nil {
		return endpoints{}, err
	}

	issuer := prm.AuthorizationServers[0]

	asm, err := a.authServerMeta(ctx, issuer)
	if err != nil {
		return endpoints{}, err
	}

	return endpoints{
		resource:        prm.Resource,
		authURL:         asm.AuthorizationEndpoint,
		tokenURL:        asm.TokenEndpoint,
		registrationURL: asm.RegistrationEndpoint,
		scopes:          prm.ScopesSupported,
	}, nil
}

func (a *Authorizer) resourceMetadata(ctx context.Context, metadataURL string) (*oauthex.ProtectedResourceMetadata, error) {
	for _, candidate := range resourceMetadataURLs(metadataURL, a.config.Key.ServerURL) {
		prm, err := oauthex.GetProtectedResourceMetadata(ctx, candidate.url, candidate.resource, a.config.HTTPClient)
		if err != nil || prm == nil {
			continue
		}

		if len(prm.AuthorizationServers) == 0 {
			return nil, errors.New("oauth: protected resource metadata has no authorization servers")
		}

		return prm, nil
	}

	return nil, fmt.Errorf("oauth: no protected resource metadata for %q", a.config.Key.ServerURL)
}

func (a *Authorizer) authServerMeta(ctx context.Context, issuer string) (*oauthex.AuthServerMeta, error) {
	for _, metadataURL := range authServerMetadataURLs(issuer) {
		asm, err := oauthex.GetAuthServerMeta(ctx, metadataURL, issuer, a.config.HTTPClient)
		if err != nil {
			return nil, err
		}

		if asm != nil {
			return asm, nil
		}
	}

	return nil, fmt.Errorf("oauth: no authorization server metadata for issuer %q", issuer)
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
	verifier := oauth2.GenerateVerifier()

	state, err := randomState()
	if err != nil {
		return nil, err
	}

	resourceParam := oauth2.SetAuthURLParam("resource", resource)

	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier), resourceParam)

	code, returnedState, err := a.config.Callback.Wait(ctx, authURL)
	if err != nil {
		return nil, err
	}

	if returnedState != state {
		return nil, errors.New("oauth: authorization state mismatch")
	}

	return cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier), resourceParam)
}

func (a *Authorizer) scopes(discovered []string) []string {
	if len(a.config.Scopes) > 0 {
		return a.config.Scopes
	}

	return discovered
}

func readChallenge(resp *http.Response) string {
	defer closeResponse(resp)

	if resp == nil {
		return ""
	}

	challenges, err := oauthex.ParseWWWAuthenticate(resp.Header.Values("WWW-Authenticate"))
	if err != nil {
		return ""
	}

	for _, challenge := range challenges {
		if challenge.Scheme != "bearer" {
			continue
		}

		if url := challenge.Params["resource_metadata"]; url != "" {
			return url
		}
	}

	return ""
}

type resourceCandidate struct {
	url      string
	resource string
}

func resourceMetadataURLs(metadataURL, resourceURL string) []resourceCandidate {
	var candidates []resourceCandidate

	if metadataURL != "" {
		candidates = append(candidates, resourceCandidate{url: metadataURL, resource: resourceURL})
	}

	rurl, err := url.Parse(resourceURL)
	if err != nil {
		return candidates
	}

	atPath := *rurl
	atPath.Path = "/.well-known/oauth-protected-resource/" + strings.TrimLeft(rurl.Path, "/")
	candidates = append(candidates, resourceCandidate{url: atPath.String(), resource: resourceURL})

	atRoot := *rurl
	atRoot.Path = "/.well-known/oauth-protected-resource"
	rurl.Path = ""
	candidates = append(candidates, resourceCandidate{url: atRoot.String(), resource: rurl.String()})

	return candidates
}

func authServerMetadataURLs(issuer string) []string {
	u, err := url.Parse(issuer)
	if err != nil {
		return nil
	}

	path := strings.TrimSuffix(u.Path, "/")

	rfc8414 := *u
	rfc8414.Path = "/.well-known/oauth-authorization-server" + path

	oidc := *u
	oidc.Path = path + "/.well-known/openid-configuration"

	return []string{rfc8414.String(), oidc.String()}
}

func randomState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func closeResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)
}
