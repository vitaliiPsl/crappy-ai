package server

import "github.com/vitaliiPsl/crappy-ai/internal/mcp"

func (s *Server) GetMCPClientStates() []mcp.ClientState {
	if s.mcpManager == nil {
		return nil
	}

	return s.mcpManager.States()
}
