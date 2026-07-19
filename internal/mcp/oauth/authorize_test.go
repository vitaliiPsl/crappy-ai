package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
)

func TestResourceMetadataURLs(t *testing.T) {
	tests := []struct {
		name        string
		metadataURL string
		resourceURL string
		want        []resourceCandidate
	}{
		{
			name:        "challenge url takes precedence then well-known fallbacks",
			metadataURL: "https://meta.example/foo",
			resourceURL: "https://api.example.com/mcp",
			want: []resourceCandidate{
				{url: "https://meta.example/foo", resource: "https://api.example.com/mcp"},
				{url: "https://api.example.com/.well-known/oauth-protected-resource/mcp", resource: "https://api.example.com/mcp"},
				{url: "https://api.example.com/.well-known/oauth-protected-resource", resource: "https://api.example.com"},
			},
		},
		{
			name:        "no challenge url falls back to well-known on the resource host",
			resourceURL: "https://api.example.com",
			want: []resourceCandidate{
				{url: "https://api.example.com/.well-known/oauth-protected-resource/", resource: "https://api.example.com"},
				{url: "https://api.example.com/.well-known/oauth-protected-resource", resource: "https://api.example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resourceMetadataURLs(tt.metadataURL, tt.resourceURL)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("resourceMetadataURLs() =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}

func TestAuthServerMetadataURLs(t *testing.T) {
	tests := []struct {
		name   string
		issuer string
		want   []string
	}{
		{
			name:   "root issuer",
			issuer: "https://as.example.com",
			want: []string{
				"https://as.example.com/.well-known/oauth-authorization-server",
				"https://as.example.com/.well-known/openid-configuration",
			},
		},
		{
			name:   "issuer with path",
			issuer: "https://as.example.com/tenant1",
			want: []string{
				"https://as.example.com/.well-known/oauth-authorization-server/tenant1",
				"https://as.example.com/tenant1/.well-known/openid-configuration",
			},
		},
		{
			name:   "trailing slash is trimmed",
			issuer: "https://as.example.com/tenant1/",
			want: []string{
				"https://as.example.com/.well-known/oauth-authorization-server/tenant1",
				"https://as.example.com/tenant1/.well-known/openid-configuration",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := authServerMetadataURLs(tt.issuer)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("authServerMetadataURLs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadChallenge(t *testing.T) {
	const want = "https://rs.example/.well-known/oauth-protected-resource"

	tests := []struct {
		name   string
		resp   *http.Response
		expect string
	}{
		{name: "nil response", resp: nil},
		{name: "no header", resp: &http.Response{Header: http.Header{}}},
		{
			name: "bearer with resource metadata",
			resp: &http.Response{Header: http.Header{
				"Www-Authenticate": {`Bearer resource_metadata="` + want + `"`},
			}},
			expect: want,
		},
		{
			name: "non-bearer scheme ignored",
			resp: &http.Response{Header: http.Header{
				"Www-Authenticate": {`Basic realm="x"`},
			}},
		},
		{
			name: "bearer without resource metadata param",
			resp: &http.Response{Header: http.Header{
				"Www-Authenticate": {`Bearer error="invalid_token"`},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readChallenge(tt.resp); got != tt.expect {
				t.Fatalf("readChallenge() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestAuthorizerScopes(t *testing.T) {
	configured := NewAuthorizer(AuthorizerConfig{Scopes: []string{"configured"}})
	if got := configured.scopes([]string{"discovered"}); !slices.Equal(got, []string{"configured"}) {
		t.Fatalf("scopes() = %v, want configured override", got)
	}

	discovered := NewAuthorizer(AuthorizerConfig{})
	if got := discovered.scopes([]string{"discovered"}); !slices.Equal(got, []string{"discovered"}) {
		t.Fatalf("scopes() = %v, want discovered fallback", got)
	}
}

func TestAuthorizerClientUsesConfiguredCredentials(t *testing.T) {
	a := NewAuthorizer(AuthorizerConfig{
		Registration: RegistrationInfo{ClientID: "cid", ClientSecret: "secret"},
	})

	id, secret, err := a.client(context.Background(), "https://as.example/register")
	if err != nil {
		t.Fatalf("client() error = %v", err)
	}

	if id != "cid" || secret != "secret" {
		t.Fatalf("client() = (%q, %q), want configured credentials", id, secret)
	}
}

func TestAuthorizerClientErrorsWithoutRegistrationEndpoint(t *testing.T) {
	a := NewAuthorizer(AuthorizerConfig{})

	if _, _, err := a.client(context.Background(), ""); err == nil {
		t.Fatal("client() error = nil, want failure without registration endpoint or client_id")
	}
}

func TestAuthServerMetaFallsBackToOIDC(t *testing.T) {
	var server *httptest.Server

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			http.Error(w, "boom", http.StatusInternalServerError)
		case "/.well-known/openid-configuration":
			writeJSON(w, authServerMetadata(server.URL))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	authorizer := NewAuthorizer(AuthorizerConfig{HTTPClient: server.Client()})

	asm, err := authorizer.authServerMeta(context.Background(), []string{server.URL})
	if err != nil {
		t.Fatalf("authServerMeta() error = %v", err)
	}

	if asm.TokenEndpoint != server.URL+"/token" {
		t.Fatalf("TokenEndpoint = %q, want %q", asm.TokenEndpoint, server.URL+"/token")
	}
}

func TestAuthServerMetaReturnsLastErrorWhenAllFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	authorizer := NewAuthorizer(AuthorizerConfig{HTTPClient: server.Client()})

	if _, err := authorizer.authServerMeta(context.Background(), []string{server.URL}); err == nil {
		t.Fatal("authServerMeta() error = nil, want failure when every candidate fails")
	}
}

func TestResourceMetadataRequiresAuthorizationServers(t *testing.T) {
	var server *httptest.Server

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"resource":              server.URL,
			"authorization_servers": []string{},
		})
	}))
	t.Cleanup(server.Close)

	authorizer := NewAuthorizer(AuthorizerConfig{
		Key:        Key{ServerURL: server.URL},
		HTTPClient: server.Client(),
	})

	if _, err := authorizer.resourceMetadata(context.Background(), ""); err == nil {
		t.Fatal("resourceMetadata() error = nil, want failure when no authorization servers are listed")
	}
}

func TestDiscover(t *testing.T) {
	server := newAuthServer(t)

	authorizer := NewAuthorizer(AuthorizerConfig{
		Key:        Key{ServerURL: server.URL},
		HTTPClient: server.Client(),
	})

	got, err := authorizer.discover(context.Background(), "")
	if err != nil {
		t.Fatalf("discover() error = %v", err)
	}

	want := endpoints{
		resource:        server.URL,
		authURL:         server.URL + "/authorize",
		tokenURL:        server.URL + "/token",
		registrationURL: server.URL + "/register",
		scopes:          []string{"read", "write"},
	}

	if got.resource != want.resource || got.authURL != want.authURL || got.tokenURL != want.tokenURL ||
		got.registrationURL != want.registrationURL || !slices.Equal(got.scopes, want.scopes) {
		t.Fatalf("discover() = %+v, want %+v", got, want)
	}
}

func TestAuthorize(t *testing.T) {
	server := newAuthServer(t)
	callback := &fakeCallback{}

	authorizer := NewAuthorizer(AuthorizerConfig{
		Key:          Key{ServerURL: server.URL},
		RedirectURL:  "http://127.0.0.1:14545/oauth/callback",
		Callback:     callback,
		HTTPClient:   server.Client(),
		Registration: RegistrationInfo{ClientID: "preconfigured-client"},
	})

	session, err := authorizer.Authorize(context.Background(), nil)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}

	if session.ClientID != "preconfigured-client" {
		t.Fatalf("ClientID = %q, want preconfigured-client", session.ClientID)
	}

	if session.Resource != server.URL || session.TokenURL != server.URL+"/token" {
		t.Fatalf("session endpoints = %+v, want resource/token at %q", session, server.URL)
	}

	if session.Token.AccessToken != "test-access" || session.Token.RefreshToken != "test-refresh" {
		t.Fatalf("token = %+v, want test-access/test-refresh", session.Token)
	}

	authURL, err := url.Parse(callback.authURL)
	if err != nil {
		t.Fatalf("callback authURL parse error = %v", err)
	}

	query := authURL.Query()
	if query.Get("resource") != server.URL {
		t.Fatalf("resource param = %q, want %q", query.Get("resource"), server.URL)
	}

	if query.Get("code_challenge") == "" || query.Get("code_challenge_method") != "S256" {
		t.Fatalf("authURL missing PKCE challenge: %v", query)
	}
}

type fakeCallback struct {
	authURL string
}

func (c *fakeCallback) Wait(_ context.Context, authURL string, _ string) (string, string, error) {
	c.authURL = authURL

	parsed, err := url.Parse(authURL)
	if err != nil {
		return "", "", err
	}

	return "test-code", parsed.Query().Get("state"), nil
}

func newAuthServer(t *testing.T) *httptest.Server {
	t.Helper()

	var server *httptest.Server

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/.well-known/oauth-protected-resource"):
			writeJSON(w, map[string]any{
				"resource":              server.URL,
				"authorization_servers": []string{server.URL},
				"scopes_supported":      []string{"read", "write"},
			})
		case r.URL.Path == "/.well-known/oauth-authorization-server":
			writeJSON(w, authServerMetadata(server.URL))
		case r.URL.Path == "/token":
			writeJSON(w, map[string]any{
				"access_token":  "test-access",
				"refresh_token": "test-refresh",
				"token_type":    "Bearer",
				"expires_in":    3600,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

func authServerMetadata(issuer string) map[string]any {
	return map[string]any{
		"issuer":                           issuer,
		"authorization_endpoint":           issuer + "/authorize",
		"token_endpoint":                   issuer + "/token",
		"registration_endpoint":            issuer + "/register",
		"jwks_uri":                         issuer + "/jwks",
		"code_challenge_methods_supported": []string{"S256"},
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
