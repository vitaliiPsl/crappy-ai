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

	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

const (
	DriverID = "openai-codex"

	accountIDMetadata = "account_id"
	defaultTokenTTL   = time.Hour
)

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

type Provider struct {
	httpClient *http.Client
}

func New() *Provider {
	return &Provider{
		httpClient: http.DefaultClient,
	}
}

func (p *Provider) ID() string {
	return DriverID
}

func (p *Provider) Authenticate(
	ctx context.Context,
	callback appoauth.Callback,
	config provideroauth.Config,
) (provideroauth.Credential, error) {
	if callback == nil {
		return provideroauth.Credential{}, errors.New("openai codex oauth: browser callback is not configured")
	}

	if err := validateConfig(config); err != nil {
		return provideroauth.Credential{}, err
	}

	token, err := p.codeFlow(config).Authorize(
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

func (p *Provider) Refresh(
	ctx context.Context,
	credential provideroauth.Credential,
	config provideroauth.Config,
) (provideroauth.Credential, error) {
	if credential.RefreshToken == "" {
		return provideroauth.Credential{}, provideroauth.ErrInvalidGrant
	}

	if err := validateConfig(config); err != nil {
		return provideroauth.Credential{}, err
	}

	token := &oauth2.Token{
		AccessToken:  credential.AccessToken,
		RefreshToken: credential.RefreshToken,
		Expiry:       time.Now().Add(-time.Second),
	}

	refreshed, err := p.codeFlow(config).TokenSource(ctx, token).Token()
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

func (p *Provider) codeFlow(config provideroauth.Config) appoauth.CodeFlow {
	return appoauth.CodeFlow{
		Config: oauth2.Config{
			ClientID:    config.ClientID,
			RedirectURL: config.RedirectURL,
			Scopes:      config.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:   config.AuthorizationURL,
				TokenURL:  config.TokenURL,
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		HTTPClient: p.httpClient,
	}
}

func validateConfig(config provideroauth.Config) error {
	switch {
	case config.ClientID == "":
		return errors.New("openai codex oauth: client_id is required")
	case config.AuthorizationURL == "":
		return errors.New("openai codex oauth: authorization_url is required")
	case config.TokenURL == "":
		return errors.New("openai codex oauth: token_url is required")
	case config.RedirectURL == "":
		return errors.New("openai codex oauth: redirect_url is required")
	default:
		return nil
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
