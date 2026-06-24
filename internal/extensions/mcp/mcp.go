package mcp

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

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

func (e *ext) Options(ctx context.Context, _ factory.BuildRequest) ([]kit.Tool, []agent.Option, error) {
	var tools []kit.Tool
	for _, client := range e.manager.List() {
		if client.State().Status != mcpcore.ClientConnected {
			continue
		}

		clientTools, err := client.ListTools(ctx)
		if err != nil {
			continue
		}

		tools = append(tools, clientTools...)
	}

	return tools, nil, nil
}
