package oauth

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
)

const (
	StatusDisconnected Status = "disconnected"
	StatusConnected    Status = "connected"
	StatusExpired      Status = "expired"
)

type Status string

type Snapshot struct {
	ProviderID string
	Status     Status
	ExpiresAt  time.Time
}

type Manager struct {
	store     Store
	callback  appoauth.Callback
	providers map[string]Provider
	sources   map[string]*source
	mu        sync.Mutex
}

func NewManager(store Store, callback appoauth.Callback, providers map[string]Provider) *Manager {
	registered := make(map[string]Provider, len(providers))
	maps.Copy(registered, providers)

	return &Manager{
		store:     store,
		callback:  callback,
		providers: registered,
		sources:   make(map[string]*source),
	}
}

func (m *Manager) Authenticate(ctx context.Context, providerID, driverID string, config Config) (Authorization, error) {
	provider, err := m.provider(driverID)
	if err != nil {
		return Authorization{}, err
	}

	source := m.source(providerID, driverID, provider)
	source.mu.Lock()
	defer source.mu.Unlock()

	credential, err := provider.Authenticate(ctx, m.callback, config)
	if err != nil {
		return Authorization{}, err
	}

	if credential.AccessToken == "" {
		return Authorization{}, errors.New("provider oauth: authentication returned an empty access token")
	}

	if err := m.store.Save(ctx, providerID, credential); err != nil {
		return Authorization{}, err
	}

	return provider.Authorization(credential), nil
}

func (m *Manager) Resolve(ctx context.Context, providerID, driverID string, config Config) (Authorization, error) {
	provider, err := m.provider(driverID)
	if err != nil {
		return Authorization{}, err
	}

	return m.source(providerID, driverID, provider).resolve(ctx, config)
}

func (m *Manager) Logout(ctx context.Context, providerID, driverID string) error {
	provider, err := m.provider(driverID)
	if err != nil {
		return err
	}

	source := m.source(providerID, driverID, provider)
	source.mu.Lock()
	defer source.mu.Unlock()

	return m.store.Delete(ctx, providerID)
}

func (m *Manager) Status(ctx context.Context, providerID, driverID string) (Snapshot, error) {
	if _, err := m.provider(driverID); err != nil {
		return Snapshot{}, err
	}

	credential, err := m.store.Load(ctx, providerID)
	if err != nil {
		return Snapshot{}, err
	}

	snapshot := Snapshot{ProviderID: providerID, Status: StatusDisconnected}
	if credential == nil || credential.AccessToken == "" {
		return snapshot, nil
	}

	snapshot.ExpiresAt = credential.ExpiresAt

	snapshot.Status = StatusConnected
	if !credential.ExpiresAt.IsZero() && time.Now().After(credential.ExpiresAt) {
		snapshot.Status = StatusExpired
	}

	return snapshot, nil
}

func (m *Manager) Supports(driverID string) bool {
	_, ok := m.providers[driverID]

	return ok
}

func (m *Manager) provider(driverID string) (Provider, error) {
	provider, ok := m.providers[driverID]
	if !ok {
		return nil, fmt.Errorf("provider oauth: unknown driver %q", driverID)
	}

	return provider, nil
}

func (m *Manager) source(providerID, driverID string, provider Provider) *source {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := providerID + "\x00" + driverID
	if source, ok := m.sources[key]; ok {
		return source
	}

	source := newSource(providerID, provider, m.store)
	m.sources[key] = source

	return source
}
