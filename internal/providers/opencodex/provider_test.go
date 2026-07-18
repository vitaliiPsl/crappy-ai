package opencodex

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

func TestAuthenticateExchangesCodeAndExtractsAccountID(t *testing.T) {
	var form url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := req.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}

		form = req.Form
		_, _ = fmt.Fprintf(w, "{\"id_token\":%q,\"access_token\":\"access\",\"refresh_token\":\"refresh\",\"expires_in\":3600}", testJWT("{\"chatgpt_account_id\":\"account\"}"))
	}))
	defer server.Close()

	provider := New()
	provider.httpClient = server.Client()
	provider.issuer = server.URL
	callback := &testCallback{}

	credential, err := provider.Authenticate(context.Background(), callback)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if credential.AccessToken != "access" || credential.RefreshToken != "refresh" {
		t.Fatalf("credential = %+v", credential)
	}

	if credential.Metadata[accountIDMetadata] != "account" {
		t.Fatalf("account ID = %q, want account", credential.Metadata[accountIDMetadata])
	}

	if form.Get("grant_type") != "authorization_code" || form.Get("code") != "code" || form.Get("code_verifier") == "" {
		t.Fatalf("token form = %v", form)
	}

	if callback.query.Get("code_challenge_method") != "S256" || callback.query.Get("codex_cli_simplified_flow") != "true" {
		t.Fatalf("authorize query = %v", callback.query)
	}
}

func TestRefreshPreservesRefreshTokenAndMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "{\"access_token\":\"new\",\"expires_in\":3600}")
	}))
	defer server.Close()

	provider := New()
	provider.httpClient = server.Client()
	provider.issuer = server.URL

	credential, err := provider.Refresh(context.Background(), provideroauth.Credential{
		AccessToken:  "old",
		RefreshToken: "refresh",
		Metadata:     map[string]string{accountIDMetadata: "account"},
	})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if credential.AccessToken != "new" || credential.RefreshToken != "refresh" || credential.Metadata[accountIDMetadata] != "account" {
		t.Fatalf("credential = %+v", credential)
	}
}

func TestRefreshClassifiesInvalidGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, "{\"error\":\"invalid_grant\",\"error_description\":\"Grant not found\"}")
	}))
	defer server.Close()

	provider := New()
	provider.httpClient = server.Client()
	provider.issuer = server.URL

	_, err := provider.Refresh(context.Background(), provideroauth.Credential{RefreshToken: "refresh"})
	if !errors.Is(err, provideroauth.ErrInvalidGrant) {
		t.Fatalf("Refresh() error = %v, want invalid grant", err)
	}
}

func TestAuthorizationIncludesCodexHeaders(t *testing.T) {
	auth := New().Authorization(provideroauth.Credential{
		AccessToken: "access",
		Metadata:    map[string]string{accountIDMetadata: "account"},
	})

	if auth.BearerToken != "access" || auth.Headers["ChatGPT-Account-Id"] != "account" || auth.Headers["originator"] != "crappy" {
		t.Fatalf("Authorization() = %+v", auth)
	}
}

type testCallback struct {
	query url.Values
}

func (c *testCallback) Wait(_ context.Context, authURL, _ string) (string, string, error) {
	parsed, err := url.Parse(authURL)
	if err != nil {
		return "", "", err
	}

	c.query = parsed.Query()

	return "code", c.query.Get("state"), nil
}

func testJWT(payload string) string {
	return strings.Join([]string{
		base64.RawURLEncoding.EncodeToString([]byte("{\"alg\":\"none\"}")),
		base64.RawURLEncoding.EncodeToString([]byte(payload)),
		"signature",
	}, ".")
}
