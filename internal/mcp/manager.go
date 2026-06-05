package mcp

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth"
)

type Manager struct {
	mu            sync.RWMutex
	clients       map[string]Client
	newClient     ClientFactory
	authenticator Authenticator
}

type ClientFactory func(Config) Client

func New(configs []Config, oauthSessionStore oauth.Store, oauthCallback oauth.Callback) *Manager {
	transport := NewTransportFactory(oauthSessionStore, nil)
	factory := func(cfg Config) Client {
		return NewClient(cfg, transport)
	}

	clients := make(map[string]Client, len(configs))
	for _, cfg := range configs {
		clients[cfg.Name] = factory(cfg)
	}

	return &Manager{
		clients:       clients,
		newClient:     factory,
		authenticator: NewOAuthAuthenticator(oauthSessionStore, oauthCallback),
	}
}

func (m *Manager) Connect(ctx context.Context) error {
	clients := m.sorted()
	errs := make([]error, len(clients))

	var wg sync.WaitGroup
	for i, client := range clients {
		if client.Config().IsEnabled() {
			wg.Go(func() {
				errs[i] = client.Connect(ctx)
			})
		}
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (m *Manager) Reconnect(ctx context.Context, name string) error {
	client, ok := m.client(name)
	if !ok {
		return fmt.Errorf("mcp: unknown client %q", name)
	}

	if err := client.Close(); err != nil {
		return err
	}

	return client.Connect(ctx)
}

func (m *Manager) Authenticate(ctx context.Context, name string) error {
	client, ok := m.client(name)
	if !ok {
		return fmt.Errorf("mcp: unknown client %q", name)
	}

	if err := m.authenticator.Authenticate(ctx, client.Config()); err != nil {
		return err
	}

	return m.ApplyConfig(ctx, client.Config())
}

func (m *Manager) ApplyConfig(ctx context.Context, config Config) error {
	client, ok := m.client(config.Name)
	if !ok {
		return fmt.Errorf("mcp: unknown client %q", config.Name)
	}

	next := m.newClient(config)

	if err := client.Close(); err != nil {
		return err
	}

	m.mu.Lock()
	m.clients[config.Name] = next
	m.mu.Unlock()

	if !config.IsEnabled() {
		return nil
	}

	return next.Connect(ctx)
}

func (m *Manager) Close() error {
	var errs []error
	for _, client := range m.Clients() {
		if err := client.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (m *Manager) Clients() []Client {
	return m.sorted()
}

func (m *Manager) Snapshots() []ClientSnapshot {
	clients := m.sorted()

	snapshots := make([]ClientSnapshot, len(clients))
	for i, client := range clients {
		snapshots[i] = ClientSnapshot{
			Config: client.Config(),
			State:  client.State(),
		}
	}

	return snapshots
}

func (m *Manager) sorted() []Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}

	sort.Slice(clients, func(i, j int) bool {
		return clients[i].Config().Name < clients[j].Config().Name
	})

	return clients
}

func (m *Manager) client(name string) (Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[name]

	return client, ok
}
