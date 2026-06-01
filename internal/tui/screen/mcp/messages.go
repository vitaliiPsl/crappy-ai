package mcp

import coremcp "github.com/vitaliiPsl/crappy-ai/internal/mcp"

type ClosedMsg struct{}

type statesLoadedMsg struct {
	states []coremcp.ClientState
}
