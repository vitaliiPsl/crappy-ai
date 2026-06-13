package mcp

import (
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
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

func (e *ext) Spec(ctx extension.Context) (spec.AgentSpec, error) {
	var tools []spec.ToolSpec
	for _, client := range e.manager.List() {
		if client.State().Status != mcpcore.ClientConnected {
			continue
		}

		clientTools, err := client.ListTools(ctx.Ctx)
		if err != nil {
			continue
		}

		for _, t := range clientTools {
			tools = append(tools, spec.ToolSpec{
				Source: "mcp:" + client.Config().Name,
				Tool:   t,
			})
		}
	}

	return spec.AgentSpec{
		Tools: tools,
	}, nil
}
