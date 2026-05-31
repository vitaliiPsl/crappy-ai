package mcp

import coremcp "github.com/vitaliiPsl/crappy-ai/internal/mcp"

type ClosedMsg struct{}

type statusesLoadedMsg struct {
	statuses []coremcp.ClientStatus
}
