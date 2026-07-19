package oauth

import (
	"context"

	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
)

type Config struct {
	ClientID         string   `yaml:"client_id,omitempty"`
	AuthorizationURL string   `yaml:"authorization_url,omitempty"`
	TokenURL         string   `yaml:"token_url,omitempty"`
	RedirectURL      string   `yaml:"redirect_url,omitempty"`
	Scopes           []string `yaml:"scopes,omitempty"`
}

type Provider interface {
	Authenticate(ctx context.Context, callback appoauth.Callback, config Config) (Credential, error)
	Refresh(ctx context.Context, credential Credential, config Config) (Credential, error)
	Authorization(credential Credential) Authorization
}

type Store interface {
	Load(ctx context.Context, providerID string) (*Credential, error)
	Save(ctx context.Context, providerID string, credential Credential) error
	Delete(ctx context.Context, providerID string) error
}
