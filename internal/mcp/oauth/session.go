package oauth

import (
	"time"

	"golang.org/x/oauth2"
)

type Key struct {
	ServerName string
	ServerURL  string
}

func NewKey(serverName, serverURL string) Key {
	return Key{ServerName: serverName, ServerURL: serverURL}
}

func (k Key) ID() string {
	return k.ServerName + "|" + k.ServerURL
}

type Session struct {
	ServerURL string `json:"server_url"`
	Resource  string `json:"resource,omitempty"`

	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`

	AuthURL  string   `json:"auth_url"`
	TokenURL string   `json:"token_url"`
	Scopes   []string `json:"scopes,omitempty"`

	Token Token `json:"token"`
}

type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitzero"`
}

func (s Session) oauthConfig(redirectURL string) oauth2.Config {
	return oauth2.Config{
		ClientID:     s.ClientID,
		ClientSecret: s.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       s.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  s.AuthURL,
			TokenURL: s.TokenURL,
		},
	}
}

func (s Session) oauthToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  s.Token.AccessToken,
		RefreshToken: s.Token.RefreshToken,
		TokenType:    s.Token.TokenType,
		Expiry:       s.Token.ExpiresAt,
	}
}

func (s Session) hasToken() bool {
	return s.Token.AccessToken != "" || s.Token.RefreshToken != ""
}

func withToken(base Session, token *oauth2.Token) Session {
	base.Token = Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresAt:    token.Expiry,
	}

	return base
}
