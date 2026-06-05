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
	mu        sync.RWMutex
	clients   map[string]Client
	transport TransportFactory
}

type Options struct {
	OAuthSessionStore oauth.Store
	OAuthCallback     oauth.Callback
}

type Option func(*Options)

func WithOAuthSessionStore(store oauth.Store) Option {
	return func(options *Options) {
		options.OAuthSessionStore = store
	}
}

func WithOAuthCallback(callback oauth.Callback) Option {
	return func(options *Options) {
		options.OAuthCallback = callback
	}
}

func New(configs []Config, opts ...Option) *Manager {
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	transport := NewTransportFactory(options)

	clients := make(map[string]Client, len(configs))
	for _, cfg := range configs {
		clients[cfg.Name] = NewClient(cfg, transport)
	}

	return &Manager{
		clients:   clients,
		transport: transport,
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

	return client.Authenticate(ctx)
}

func (m *Manager) SetEnabled(ctx context.Context, name string, enabled bool) error {
	client, ok := m.client(name)
	if !ok {
		return fmt.Errorf("mcp: unknown client %q", name)
	}

	cfg := client.Config()
	if cfg.IsEnabled() == enabled {
		return nil
	}

	cfg.Enabled = &enabled
	next := NewClient(cfg, m.transport)

	if err := client.Close(); err != nil {
		return err
	}

	m.mu.Lock()
	m.clients[name] = next
	m.mu.Unlock()

	if !enabled {
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

func (m *Manager) Configs() []Config {
	clients := m.sorted()

	configs := make([]Config, len(clients))
	for i, client := range clients {
		configs[i] = client.Config()
	}

	return configs
}

func (m *Manager) States() []ClientState {
	clients := m.sorted()

	states := make([]ClientState, len(clients))
	for i, client := range clients {
		states[i] = client.State()
	}

	return states
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
