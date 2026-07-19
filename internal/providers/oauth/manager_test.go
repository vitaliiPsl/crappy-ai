package oauth

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
)

func TestManagerAuthenticateAndResolve(t *testing.T) {
	credential := Credential{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}
	provider := &fakeProvider{authenticated: credential}
	store := newFakeStore()
	manager := NewManager(store, fakeCallback{}, map[string]Provider{"openai": provider})

	config := Config{ClientID: "client"}

	auth, err := manager.Authenticate(context.Background(), "openai", "openai", config)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if auth.BearerToken != "access" {
		t.Fatalf("Authenticate() token = %q, want access", auth.BearerToken)
	}

	auth, err = manager.Resolve(context.Background(), "openai", "openai", config)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if auth.BearerToken != "access" || provider.refreshes.Load() != 0 {
		t.Fatalf("Resolve() = %+v, refreshes = %d", auth, provider.refreshes.Load())
	}

	if provider.config.ClientID != config.ClientID {
		t.Fatalf("Authenticate() config = %+v, want %+v", provider.config, config)
	}
}

func TestManagerResolveRefreshesOnce(t *testing.T) {
	store := newFakeStore()
	store.credentials["openai"] = Credential{
		AccessToken:  "old",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Minute),
	}
	provider := &fakeProvider{refreshed: Credential{AccessToken: "new", ExpiresAt: time.Now().Add(time.Hour)}}
	manager := NewManager(store, nil, map[string]Provider{"openai": provider})

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			auth, err := manager.Resolve(context.Background(), "openai", "openai", Config{})
			if err != nil {
				t.Errorf("Resolve() error = %v", err)
			} else if auth.BearerToken != "new" {
				t.Errorf("Resolve() token = %q, want new", auth.BearerToken)
			}
		})
	}

	wg.Wait()

	if got := provider.refreshes.Load(); got != 1 {
		t.Fatalf("Refresh() calls = %d, want 1", got)
	}
}

func TestManagerResolveInvalidGrantDeletesCredential(t *testing.T) {
	store := newFakeStore()
	store.credentials["openai"] = Credential{AccessToken: "old", ExpiresAt: time.Now().Add(-time.Minute)}
	provider := &fakeProvider{refreshErr: ErrInvalidGrant}
	manager := NewManager(store, nil, map[string]Provider{"openai": provider})

	_, err := manager.Resolve(context.Background(), "openai", "openai", Config{})
	if !errors.Is(err, ErrAuthRequired) || !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("Resolve() error = %v, want auth required and invalid grant", err)
	}

	if _, ok := store.credentials["openai"]; ok {
		t.Fatal("credential remains after invalid grant")
	}
}

func TestManagerResolveKeepsCredentialAfterTransientFailure(t *testing.T) {
	wantErr := errors.New("temporarily unavailable")
	store := newFakeStore()
	store.credentials["openai"] = Credential{AccessToken: "old", ExpiresAt: time.Now().Add(-time.Minute)}
	manager := NewManager(store, nil, map[string]Provider{
		"openai": &fakeProvider{refreshErr: wantErr},
	})

	_, err := manager.Resolve(context.Background(), "openai", "openai", Config{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Resolve() error = %v, want %v", err, wantErr)
	}

	if _, ok := store.credentials["openai"]; !ok {
		t.Fatal("credential deleted after transient failure")
	}
}

func TestManagerStatusAndLogout(t *testing.T) {
	store := newFakeStore()
	manager := NewManager(store, nil, map[string]Provider{"openai": &fakeProvider{}})

	snapshot, err := manager.Status(context.Background(), "openai", "openai")
	if err != nil || snapshot.Status != StatusDisconnected {
		t.Fatalf("Status() = %+v, %v", snapshot, err)
	}

	store.credentials["openai"] = Credential{AccessToken: "access", ExpiresAt: time.Now().Add(-time.Minute)}

	snapshot, err = manager.Status(context.Background(), "openai", "openai")
	if err != nil || snapshot.Status != StatusExpired {
		t.Fatalf("Status() = %+v, %v", snapshot, err)
	}

	if err := manager.Logout(context.Background(), "openai", "openai"); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if _, ok := store.credentials["openai"]; ok {
		t.Fatal("credential remains after Logout()")
	}
}

type fakeCallback struct{}

func (fakeCallback) Wait(context.Context, string, string) (string, string, error) {
	return "code", "state", nil
}

var _ appoauth.Callback = fakeCallback{}

type fakeProvider struct {
	authenticated Credential
	refreshed     Credential
	refreshErr    error
	refreshes     atomic.Int32
	config        Config
}

func (p *fakeProvider) Authenticate(_ context.Context, _ appoauth.Callback, config Config) (Credential, error) {
	p.config = config

	return p.authenticated, nil
}

func (p *fakeProvider) Refresh(_ context.Context, _ Credential, config Config) (Credential, error) {
	p.refreshes.Add(1)
	p.config = config

	return p.refreshed, p.refreshErr
}

func (p *fakeProvider) Authorization(credential Credential) Authorization {
	return Authorization{BearerToken: credential.AccessToken}
}

type fakeStore struct {
	mu          sync.Mutex
	credentials map[string]Credential
}

func newFakeStore() *fakeStore {
	return &fakeStore{credentials: make(map[string]Credential)}
}

func (s *fakeStore) Load(_ context.Context, providerID string) (*Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	credential, ok := s.credentials[providerID]
	if !ok {
		return nil, nil
	}

	return &credential, nil
}

func (s *fakeStore) Save(_ context.Context, providerID string, credential Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.credentials[providerID] = credential

	return nil
}

func (s *fakeStore) Delete(_ context.Context, providerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.credentials, providerID)

	return nil
}
