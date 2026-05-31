package mcp

import (
	"github.com/vitaliiPsl/crappy-adk/agent"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
	mcpcore "github.com/vitaliiPsl/crappy-ai/internal/mcp"
)

type ext struct {
	manager *mcpcore.Manager
}

func New(manager *mcpcore.Manager) extension.Extension {
	return &ext{
		manager: manager,
	}
}

func (e *ext) Name() string {
	return "mcp"
}

func (e *ext) Options(ctx extension.Context) (agent.Option, error) {
	return agent.WithTools(e.manager.Tools(ctx.Ctx)...), nil
}
