package mcp

import (
	"context"
	"errors"
	"sync"
)

type Manager struct {
	clients []Client
}

func New(configs []Config) *Manager {
	clients := make([]Client, len(configs))
	for i, cfg := range configs {
		clients[i] = NewClient(cfg)
	}

	return NewWithClients(clients...)
}

func NewWithClients(clients ...Client) *Manager {
	return &Manager{
		clients: clients,
	}
}

func (m *Manager) Connect(ctx context.Context) error {
	errs := make([]error, len(m.clients))

	var wg sync.WaitGroup
	for i, client := range m.clients {
		if client.Config().IsEnabled() {
			wg.Go(func() {
				errs[i] = client.Connect(ctx)
			})
		}
	}

	wg.Wait()

	return errors.Join(errs...)
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
	clients := make([]Client, len(m.clients))
	copy(clients, m.clients)

	return clients
}

func (m *Manager) Configs() []Config {
	configs := make([]Config, len(m.clients))
	for i, client := range m.clients {
		configs[i] = client.Config()
	}

	return configs
}

func (m *Manager) States() []ClientState {
	states := make([]ClientState, len(m.clients))
	for i, client := range m.clients {
		states[i] = client.State()
	}

	return states
}
