package mcp

import (
	"context"
	"errors"
	"sync"

	"github.com/vitaliiPsl/crappy-adk/kit"
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
		wg.Go(func() {
			errs[i] = client.Connect(ctx)
		})
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (m *Manager) Clients() []Client {
	clients := make([]Client, len(m.clients))
	copy(clients, m.clients)

	return clients
}

func (m *Manager) Statuses() []ClientStatus {
	statuses := make([]ClientStatus, len(m.clients))
	for i, client := range m.clients {
		statuses[i] = client.Status()
	}

	return statuses
}

func (m *Manager) Tools(ctx context.Context) []kit.Tool {
	var tools []kit.Tool
	for _, client := range m.clients {
		if client.Status().State != ClientConnected {
			continue
		}

		defs, err := client.ListTools(ctx)
		if err != nil {
			continue
		}

		for _, def := range defs {
			tools = append(tools, newTool(client, def))
		}
	}

	return tools
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
