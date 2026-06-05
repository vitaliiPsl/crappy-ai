package mcp

import coremcp "github.com/vitaliiPsl/crappy-ai/internal/mcp"

type ClosedMsg struct{}

type clientsLoadedMsg struct {
	clients []coremcp.ClientSnapshot
}
