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
	clients map[string]Client
}

type Options struct {
	OAuthSessionStore oauth.SessionStore
	OAuthPrompter     oauth.Prompter
}

type Option func(*Options)

func WithOAuthSessionStore(store oauth.SessionStore) Option {
	return func(options *Options) {
		options.OAuthSessionStore = store
	}
}

func WithOAuthPrompter(prompter oauth.Prompter) Option {
	return func(options *Options) {
		options.OAuthPrompter = prompter
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
		clients: clients,
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
	client, ok := m.clients[name]
	if !ok {
		return fmt.Errorf("mcp: unknown client %q", name)
	}

	if err := client.Close(); err != nil {
		return err
	}

	return client.Connect(ctx)
}

func (m *Manager) Authenticate(ctx context.Context, name string) error {
	client, ok := m.clients[name]
	if !ok {
		return fmt.Errorf("mcp: unknown client %q", name)
	}

	return client.Authenticate(ctx)
}

func (m *Manager) Close() error {
	var errs []error
	for _, client := range m.clients {
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
	clients := make([]Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}

	sort.Slice(clients, func(i, j int) bool {
		return clients[i].Config().Name < clients[j].Config().Name
	})

	return clients
}
