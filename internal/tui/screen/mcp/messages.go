package mcp

import coremcp "github.com/vitaliiPsl/crappy-ai/internal/mcp"

type ClosedMsg struct{}

type configsLoadedMsg struct {
	configs []coremcp.Config
}

type statesLoadedMsg struct {
	states []coremcp.ClientState
}
