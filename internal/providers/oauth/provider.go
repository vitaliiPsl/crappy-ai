package oauth

import (
	"context"

	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
)

type Provider interface {
	Authenticate(ctx context.Context, callback appoauth.Callback) (Credential, error)
	Refresh(ctx context.Context, credential Credential) (Credential, error)
	Authorization(credential Credential) Authorization
}

type Store interface {
	Load(ctx context.Context, providerID string) (*Credential, error)
	Save(ctx context.Context, providerID string, credential Credential) error
	Delete(ctx context.Context, providerID string) error
}
