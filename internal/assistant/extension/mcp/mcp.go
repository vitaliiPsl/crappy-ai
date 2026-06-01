package mcp

import (
	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

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
	var tools []kit.Tool
	for _, client := range e.manager.Clients() {
		if client.State().Status != mcpcore.ClientConnected {
			continue
		}

		clientTools, err := client.ListTools(ctx.Ctx)
		if err != nil {
			continue
		}

		tools = append(tools, clientTools...)
	}

	return agent.WithTools(tools...), nil
}
