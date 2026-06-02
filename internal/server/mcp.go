package server

import (
	"context"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
)

func (s *Server) GetMCPClientConfigs() []mcp.Config {
	if s.mcpManager == nil {
		return nil
	}

	return s.mcpManager.Configs()
}

func (s *Server) GetMCPClientStates() []mcp.ClientState {
	if s.mcpManager == nil {
		return nil
	}

	return s.mcpManager.States()
}

func (s *Server) ReconnectMCPClient(ctx context.Context, name string) error {
	if s.mcpManager == nil {
		return nil
	}

	return s.mcpManager.Reconnect(ctx, name)
}