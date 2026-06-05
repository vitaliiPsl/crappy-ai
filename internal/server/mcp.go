package server

import (
	"context"
	"fmt"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
)

func (s *Server) GetMCPClientSnapshots() []mcp.ClientSnapshot {
	if s.mcpManager == nil {
		return nil
	}

	return s.mcpManager.Snapshots()
}

func (s *Server) ReconnectMCPClient(ctx context.Context, name string) error {
	if s.mcpManager == nil {
		return nil
	}

	return s.mcpManager.Reconnect(ctx, name)
}

func (s *Server) AuthenticateMCPClient(ctx context.Context, name string) error {
	if s.mcpManager == nil {
		return nil
	}

	return s.mcpManager.Authenticate(ctx, name)
}

func (s *Server) SetMCPClientEnabled(ctx context.Context, name string, enabled bool) error {
	if s.mcpManager == nil {
		return nil
	}

	settings := s.settingsStore.Get()

	var config mcp.Config

	found := false
	for i := range settings.MCPClients {
		if settings.MCPClients[i].Name != name {
			continue
		}

		settings.MCPClients[i].Enabled = &enabled
		config = settings.MCPClients[i]
		found = true

		break
	}

	if !found {
		return fmt.Errorf("mcp: unknown client %q", name)
	}

	if err := s.settingsStore.Save(settings); err != nil {
		return err
	}

	return s.mcpManager.ApplyConfig(ctx, config)
}
