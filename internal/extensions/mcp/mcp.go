package mcp

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/kit"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
	mcpcore "github.com/vitaliiPsl/crappy-ai/internal/mcp"
)

type ext struct {
	manager *mcpcore.Manager
}

type Extension interface {
	appagent.Contributor
}

var _ appagent.Contributor = (*ext)(nil)

func New(manager *mcpcore.Manager) Extension {
	return &ext{
		manager: manager,
	}
}

func (e *ext) Name() string {
	return "mcp"
}

func (e *ext) Contribute(ctx context.Context, _ appagent.Request) (appagent.Contribution, error) {
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

	return appagent.Contribution{Tools: tools}, nil
}
