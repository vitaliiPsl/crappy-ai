package opencodex

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

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

type tokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type tokenError struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
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

	verifier, err := randomString(32)
	if err != nil {
		return provideroauth.Credential{}, err
	}

	state, err := randomString(32)
	if err != nil {
		return provideroauth.Credential{}, err
	}

	code, returnedState, err := callback.Wait(ctx, p.authorizeURL(verifier, state), p.redirectURL)
	if err != nil {
		return provideroauth.Credential{}, err
	}

	if returnedState != state {
		return provideroauth.Credential{}, errors.New("openai codex oauth: authorization state mismatch")
	}

	return p.exchange(ctx, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.redirectURL},
		"client_id":     {clientID},
		"code_verifier": {verifier},
	})
}

func (p *Provider) Refresh(ctx context.Context, credential provideroauth.Credential) (provideroauth.Credential, error) {
	if credential.RefreshToken == "" {
		return provideroauth.Credential{}, provideroauth.ErrInvalidGrant
	}

	refreshed, err := p.exchange(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {credential.RefreshToken},
		"client_id":     {clientID},
	})
	if err != nil {
		return provideroauth.Credential{}, err
	}

	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = credential.RefreshToken
	}

	if len(refreshed.Metadata) == 0 {
		refreshed.Metadata = credential.Metadata
	}

	return refreshed, nil
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

func (p *Provider) authorizeURL(verifier, state string) string {
	challenge := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"response_type":              {"code"},
		"client_id":                  {clientID},
		"redirect_uri":               {p.redirectURL},
		"scope":                      {"openid profile email offline_access"},
		"code_challenge":             {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method":      {"S256"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
		"state":                      {state},
		"originator":                 {"crappy"},
	}

	return p.issuer + "/oauth/authorize?" + query.Encode()
}

func (p *Provider) exchange(ctx context.Context, form url.Values) (provideroauth.Credential, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.issuer+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return provideroauth.Credential{}, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "crappy")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return provideroauth.Credential{}, fmt.Errorf("openai codex oauth: exchange token: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return provideroauth.Credential{}, decodeTokenError(resp)
	}

	var tokens tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return provideroauth.Credential{}, fmt.Errorf("openai codex oauth: decode token: %w", err)
	}

	if tokens.AccessToken == "" {
		return provideroauth.Credential{}, errors.New("openai codex oauth: token response has no access token")
	}

	ttl := time.Duration(tokens.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}

	credential := provideroauth.Credential{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    time.Now().Add(ttl),
	}

	if accountID := extractAccountID(tokens); accountID != "" {
		credential.Metadata = map[string]string{accountIDMetadata: accountID}
	}

	return credential, nil
}

func decodeTokenError(resp *http.Response) error {
	result := tokenError{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	message := result.Description
	if message == "" {
		message = result.Code
	}

	if message == "" {
		message = resp.Status
	}

	err := fmt.Errorf("openai codex oauth: token request failed: %s", message)
	if result.Code == "invalid_grant" {
		return errors.Join(err, provideroauth.ErrInvalidGrant)
	}

	return err
}

func extractAccountID(tokens tokenResponse) string {
	if accountID := accountIDFromToken(tokens.IDToken); accountID != "" {
		return accountID
	}

	return accountIDFromToken(tokens.AccessToken)
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

func randomString(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("openai codex oauth: generate random value: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}
