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
	ctx       context.Context
	cancel    context.CancelFunc
	startOnce sync.Once

	mu            sync.RWMutex
	clients       map[string]Client
	newClient     ClientFactory
	authenticator Authenticator
}

type ClientFactory func(Config) Client

func New(configs []Config, oauthSessionStore oauth.Store, oauthCallback oauth.Callback) *Manager {
	return NewManager(context.Background(), configs, oauthSessionStore, oauthCallback)
}

func NewManager(ctx context.Context, configs []Config, oauthSessionStore oauth.Store, oauthCallback oauth.Callback) *Manager {
	ctx, cancel := context.WithCancel(ctx)

	transport := NewTransportFactory(oauthSessionStore, nil)
	factory := func(cfg Config) Client {
		return NewClient(cfg, transport)
	}

	clients := make(map[string]Client, len(configs))
	for _, cfg := range configs {
		clients[cfg.Name] = factory(cfg)
	}

	return &Manager{
		ctx:           ctx,
		cancel:        cancel,
		clients:       clients,
		newClient:     factory,
		authenticator: NewOAuthAuthenticator(oauthSessionStore, oauthCallback),
	}
}

func (m *Manager) Start() {
	m.startOnce.Do(func() {
		go func() { _ = m.Connect() }()
	})
}

func (m *Manager) Close() {
	m.cancel()

	for _, client := range m.List() {
		_ = client.Close()
	}
}

func (m *Manager) Connect() error {
	if err := m.ctx.Err(); err != nil {
		return err
	}

	clients := m.List()
	errs := make([]error, len(clients))

	var wg sync.WaitGroup
	for i, client := range clients {
		if client.Config().IsEnabled() {
			wg.Go(func() {
				errs[i] = client.Connect(m.ctx)
			})
		}
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (m *Manager) Reconnect(ctx context.Context, name string) error {
	if err := m.ctx.Err(); err != nil {
		return err
	}

	client, err := m.Get(name)
	if err != nil {
		return err
	}

	if err := client.Close(); err != nil {
		return err
	}

	return client.Connect(ctx)
}

func (m *Manager) Authenticate(ctx context.Context, name string) error {
	if err := m.ctx.Err(); err != nil {
		return err
	}

	client, err := m.Get(name)
	if err != nil {
		return err
	}

	if err := m.authenticator.Authenticate(ctx, client.Config()); err != nil {
		return err
	}

	return m.ApplyConfig(ctx, client.Config())
}

func (m *Manager) ApplyConfig(ctx context.Context, config Config) error {
	if err := m.ctx.Err(); err != nil {
		return err
	}

	client, err := m.Get(config.Name)
	if err != nil {
		return err
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

func (m *Manager) Clients() []Client {
	return m.List()
}

func (m *Manager) Snapshots() []ClientSnapshot {
	clients := m.List()

	snapshots := make([]ClientSnapshot, len(clients))
	for i, client := range clients {
		snapshots[i] = ClientSnapshot{
			Config: client.Config(),
			State:  client.State(),
		}
	}

	return snapshots
}

func (m *Manager) List() []Client {
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

func (m *Manager) Get(name string) (Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[name]
	if !ok {
		return nil, fmt.Errorf("mcp: unknown client %q", name)
	}

	return client, nil
}
