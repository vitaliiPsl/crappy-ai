package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestCodeFlowAuthorizeUsesPKCEAndValidatesState(t *testing.T) {
	var form url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := req.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}

		form = req.Form

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access",
			"token_type":   "Bearer",
		})
	}))
	t.Cleanup(server.Close)

	callback := &flowCallback{}
	flow := CodeFlow{
		Config: oauth2.Config{
			ClientID:    "client",
			RedirectURL: "http://127.0.0.1/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  server.URL + "/authorize",
				TokenURL: server.URL,
			},
		},
		HTTPClient: server.Client(),
	}

	token, err := flow.Authorize(context.Background(), callback, CodeFlowOptions{})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}

	if token.AccessToken != "access" {
		t.Fatalf("AccessToken = %q, want access", token.AccessToken)
	}

	if callback.query.Get("code_challenge") == "" || callback.query.Get("code_challenge_method") != "S256" {
		t.Fatalf("authorization query = %v, want PKCE challenge", callback.query)
	}

	if form.Get("code_verifier") == "" || form.Get("code") != "code" {
		t.Fatalf("token form = %v, want code and PKCE verifier", form)
	}
}

func TestCodeFlowAuthorizeRequiresCallback(t *testing.T) {
	_, err := (CodeFlow{}).Authorize(context.Background(), nil, CodeFlowOptions{})
	if err == nil {
		t.Fatal("Authorize() error = nil, want missing callback error")
	}
}

type flowCallback struct {
	query url.Values
}

func (c *flowCallback) Wait(_ context.Context, authURL, _ string) (string, string, error) {
	parsed, err := url.Parse(authURL)
	if err != nil {
		return "", "", err
	}

	c.query = parsed.Query()

	return "code", c.query.Get("state"), nil
}
