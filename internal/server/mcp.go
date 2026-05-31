package server

import "github.com/vitaliiPsl/crappy-ai/internal/mcp"

func (s *Server) GetMCPClientStatuses() []mcp.ClientStatus {
	if s.mcpManager == nil {
		return nil
	}

	return s.mcpManager.Statuses()
}
