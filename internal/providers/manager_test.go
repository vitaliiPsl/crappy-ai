package providers

import (
	"context"
	"testing"
	"time"

	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

func TestManagerResolvesProviderAccess(t *testing.T) {
	store := &memoryStore{credentials: map[string]provideroauth.Credential{
		"work": {AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)},
	}}
	manager := NewManager(store, nil, testProvider{})

	auth, err := manager.Resolve(context.Background(), "work", "test", provideroauth.Config{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if auth.BearerToken != "access" {
		t.Fatalf("BearerToken = %q, want access", auth.BearerToken)
	}
}

func TestManagerFetchesProviderLimits(t *testing.T) {
	store := &memoryStore{credentials: map[string]provideroauth.Credential{
		"work": {AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)},
	}}
	provider := &testLimitsProvider{testProvider: testProvider{}}
	manager := NewManager(store, nil, provider)

	limits, err := manager.Limits(
		context.Background(),
		"work",
		"test",
		provideroauth.Config{LimitsURL: "https://example.test"},
	)
	if err != nil {
		t.Fatalf("Limits() error = %v", err)
	}

	if limits.Plan != "plus" || provider.auth.BearerToken != "access" || provider.limitsURL != "https://example.test" {
		t.Fatalf("Limits() = %+v, provider = %+v", limits, provider)
	}
}

type testProvider struct{}

func (testProvider) ID() string {
	return "test"
}

func (testProvider) Authenticate(context.Context, appoauth.Callback, provideroauth.Config) (provideroauth.Credential, error) {
	return provideroauth.Credential{}, nil
}

func (testProvider) Refresh(context.Context, provideroauth.Credential, provideroauth.Config) (provideroauth.Credential, error) {
	return provideroauth.Credential{}, nil
}

func (testProvider) Authorization(credential provideroauth.Credential) provideroauth.Authorization {
	return provideroauth.Authorization{BearerToken: credential.AccessToken}
}

func (testProvider) Limits(
	context.Context,
	provideroauth.Authorization,
	provideroauth.Config,
) (provideroauth.Limits, error) {
	return provideroauth.Limits{}, nil
}

type testLimitsProvider struct {
	testProvider
	limitsURL string
	auth      provideroauth.Authorization
}

func (p *testLimitsProvider) Limits(
	_ context.Context,
	auth provideroauth.Authorization,
	config provideroauth.Config,
) (provideroauth.Limits, error) {
	p.limitsURL = config.LimitsURL
	p.auth = auth

	return provideroauth.Limits{Plan: "plus"}, nil
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
