package mcp

import (
	"context"
	"errors"

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
	var errs []error
	for _, client := range m.clients {
		if err := client.Connect(ctx); err != nil {
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

func (m *Manager) Tools(ctx context.Context) ([]kit.Tool, error) {
	var tools []kit.Tool
	for _, client := range m.clients {
		defs, err := client.ListTools(ctx)
		if err != nil {
			return nil, err
		}

		for _, def := range defs {
			tools = append(tools, newTool(client, def))
		}
	}

	return tools, nil
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
