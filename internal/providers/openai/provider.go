package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	adkproviders "github.com/vitaliiPsl/crappy-adk/providers"

	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

const (
	DriverID    = "openai-codex"
	CodexAPIURL = "https://chatgpt.com/backend-api/codex"

	clientID    = "app_EMoamEEZ73f0CkXaXp7hrann"
	issuer      = "https://auth.openai.com"
	redirectURL = "http://localhost:1455/auth/callback"

	accountIDMetadata = "account_id"
	defaultTokenTTL   = time.Hour
)

type Provider struct {
	httpClient  *http.Client
	issuer      string
	redirectURL string
}

type claims struct {
	AccountID     string         `json:"chatgpt_account_id"`
	Organizations []organization `json:"organizations"`
	APIAuth       apiAuth        `json:"https://api.openai.com/auth"`
}

type organization struct {
	ID string `json:"id"`
}

type apiAuth struct {
	AccountID string `json:"chatgpt_account_id"`
}

func New() *Provider {
	return &Provider{
		httpClient:  http.DefaultClient,
		issuer:      issuer,
		redirectURL: redirectURL,
	}
}

func (p *Provider) ID() string {
	return DriverID
}

func (p *Provider) ModelOptions(auth provideroauth.Authorization) []adkproviders.ModelOption {
	return []adkproviders.ModelOption{
		adkproviders.WithBaseURL(CodexAPIURL),
		adkproviders.WithBearerToken(auth.BearerToken),
		adkproviders.WithHeaders(auth.Headers),
	}
}

func (p *Provider) Authenticate(ctx context.Context, callback appoauth.Callback) (provideroauth.Credential, error) {
	if callback == nil {
		return provideroauth.Credential{}, errors.New("openai codex oauth: browser callback is not configured")
	}

	token, err := p.codeFlow().Authorize(
		ctx,
		callback,
		appoauth.CodeFlowOptions{Authorization: []oauth2.AuthCodeOption{
			oauth2.SetAuthURLParam("id_token_add_organizations", "true"),
			oauth2.SetAuthURLParam("codex_cli_simplified_flow", "true"),
			oauth2.SetAuthURLParam("originator", "crappy"),
		}},
	)
	if err != nil {
		return provideroauth.Credential{}, fmt.Errorf("openai codex oauth: authorize: %w", err)
	}

	return credentialFromToken(token, nil)
}

func (p *Provider) Refresh(ctx context.Context, credential provideroauth.Credential) (provideroauth.Credential, error) {
	if credential.RefreshToken == "" {
		return provideroauth.Credential{}, provideroauth.ErrInvalidGrant
	}

	token := &oauth2.Token{
		AccessToken:  credential.AccessToken,
		RefreshToken: credential.RefreshToken,
		Expiry:       time.Now().Add(-time.Second),
	}

	refreshed, err := p.codeFlow().TokenSource(ctx, token).Token()
	if err != nil {
		wrapped := fmt.Errorf("openai codex oauth: refresh token: %w", err)
		if appoauth.IsInvalidGrant(err) {
			return provideroauth.Credential{}, errors.Join(wrapped, provideroauth.ErrInvalidGrant)
		}

		return provideroauth.Credential{}, wrapped
	}

	return credentialFromToken(refreshed, credential.Metadata)
}

func (p *Provider) Authorization(credential provideroauth.Credential) provideroauth.Authorization {
	headers := map[string]string{"originator": "crappy"}
	if accountID := credential.Metadata[accountIDMetadata]; accountID != "" {
		headers["ChatGPT-Account-Id"] = accountID
	}

	return provideroauth.Authorization{
		BearerToken: credential.AccessToken,
		Headers:     headers,
	}
}

func (p *Provider) codeFlow() appoauth.CodeFlow {
	return appoauth.CodeFlow{
		Config: oauth2.Config{
			ClientID:    clientID,
			RedirectURL: p.redirectURL,
			Scopes:      []string{"openid", "profile", "email", "offline_access"},
			Endpoint: oauth2.Endpoint{
				AuthURL:   p.issuer + "/oauth/authorize",
				TokenURL:  p.issuer + "/oauth/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		HTTPClient: p.httpClient,
	}
}

func credentialFromToken(token *oauth2.Token, fallbackMetadata map[string]string) (provideroauth.Credential, error) {
	if token == nil || token.AccessToken == "" {
		return provideroauth.Credential{}, errors.New("openai codex oauth: token response has no access token")
	}

	expiresAt := token.Expiry
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(defaultTokenTTL)
	}

	credential := provideroauth.Credential{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    expiresAt,
		Metadata:     fallbackMetadata,
	}

	idToken, _ := token.Extra("id_token").(string)
	if accountID := extractAccountID(idToken, token.AccessToken); accountID != "" {
		credential.Metadata = map[string]string{accountIDMetadata: accountID}
	}

	return credential, nil
}

func extractAccountID(idToken, accessToken string) string {
	if accountID := accountIDFromToken(idToken); accountID != "" {
		return accountID
	}

	return accountIDFromToken(accessToken)
}

func accountIDFromToken(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var parsed claims
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return ""
	}

	if parsed.AccountID != "" {
		return parsed.AccountID
	}

	if parsed.APIAuth.AccountID != "" {
		return parsed.APIAuth.AccountID
	}

	if len(parsed.Organizations) > 0 {
		return parsed.Organizations[0].ID
	}

	return ""
}
