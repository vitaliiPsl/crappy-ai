package mcp

import (
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	mcpcore "github.com/vitaliiPsl/crappy-ai/internal/mcp"
)

type ext struct {
	manager *mcpcore.Manager
}

func New(manager *mcpcore.Manager) factory.Extension {
	return &ext{
		manager: manager,
	}
}

func (e *ext) Name() string {
	return "mcp"
}

func (e *ext) Spec(ctx factory.Context) (factory.AgentSpec, error) {
	var tools []factory.ToolSpec
	for _, client := range e.manager.List() {
		if client.State().Status != mcpcore.ClientConnected {
			continue
		}

		clientTools, err := client.ListTools(ctx.Ctx)
		if err != nil {
			continue
		}

		for _, t := range clientTools {
			tools = append(tools, factory.ToolSpec{
				Source: "mcp:" + client.Config().Name,
				Tool:   t,
			})
		}
	}

	return factory.AgentSpec{
		Tools: tools,
	}, nil
}
