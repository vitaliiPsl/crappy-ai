package mcp

import (
	"context"

	adk "github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	mcpcore "github.com/vitaliiPsl/crappy-ai/internal/mcp"
)

type ext struct {
	manager *mcpcore.Manager
}

type Extension interface {
	factory.Extension
	appagent.Contributor
}

var _ factory.Extension = (*ext)(nil)
var _ appagent.Contributor = (*ext)(nil)

func New(manager *mcpcore.Manager) Extension {
	return &ext{
		manager: manager,
	}
}

func (e *ext) Name() string {
	return "mcp"
}

func (e *ext) Options(ctx context.Context, _ factory.BuildRequest) ([]kit.Tool, []adk.Option, error) {
	c, err := e.Contribute(ctx, appagent.Request{})

	return c.Tools, c.Options, err
}

func (e *ext) Contribute(ctx context.Context, _ appagent.Request) (appagent.Contribution, error) {
	if e.manager == nil {
		return appagent.Contribution{}, nil
	}

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
