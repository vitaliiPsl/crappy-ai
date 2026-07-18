package providers

import (
	"context"
	"testing"
	"time"

	adkproviders "github.com/vitaliiPsl/crappy-adk/providers"

	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

func TestManagerResolvesProviderAccess(t *testing.T) {
	store := &memoryStore{credentials: map[string]provideroauth.Credential{
		"work": {AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)},
	}}
	manager := NewManager(store, nil, testProvider{})

	auth, err := manager.Resolve(context.Background(), "work", "test")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	got := adkproviders.ModelOptions{}

	options := manager.ModelOptions("test", auth)
	for _, option := range options {
		option(&got)
	}

	if got.BearerToken != "access" {
		t.Fatalf("BearerToken = %q, want access", got.BearerToken)
	}
}

type testProvider struct{}

func (testProvider) ID() string {
	return "test"
}

func (testProvider) Authenticate(context.Context, appoauth.Callback) (provideroauth.Credential, error) {
	return provideroauth.Credential{}, nil
}

func (testProvider) Refresh(context.Context, provideroauth.Credential) (provideroauth.Credential, error) {
	return provideroauth.Credential{}, nil
}

func (testProvider) Authorization(credential provideroauth.Credential) provideroauth.Authorization {
	return provideroauth.Authorization{BearerToken: credential.AccessToken}
}

func (testProvider) ModelOptions(auth provideroauth.Authorization) []adkproviders.ModelOption {
	return []adkproviders.ModelOption{adkproviders.WithBearerToken(auth.BearerToken)}
}

type memoryStore struct {
	credentials map[string]provideroauth.Credential
}

func (s *memoryStore) Load(_ context.Context, providerID string) (*provideroauth.Credential, error) {
	credential, ok := s.credentials[providerID]
	if !ok {
		return nil, nil
	}

	return &credential, nil
}

func (s *memoryStore) Save(_ context.Context, providerID string, credential provideroauth.Credential) error {
	s.credentials[providerID] = credential

	return nil
}

func (s *memoryStore) Delete(_ context.Context, providerID string) error {
	delete(s.credentials, providerID)

	return nil
}
