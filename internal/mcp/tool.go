package mcp

import (
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

var _ kit.Tool = (*tool)(nil)

type tool struct {
	name   string
	def    kit.ToolDefinition
	client Client
}

func newTool(client Client, def kit.ToolDefinition) kit.Tool {
	return &tool{
		name:   fmt.Sprintf("mcp__%s__%s", client.Config().Name, def.Name),
		def:    def,
		client: client,
	}
}

func (t *tool) Definition() kit.ToolDefinition {
	return kit.ToolDefinition{
		Name:        t.name,
		Description: t.def.Description,
		Schema:      t.def.Schema,
	}
}

func (t *tool) Execute(rc *kit.RunContext, args map[string]any) (string, error) {
	result, err := t.client.CallTool(rc.Context, kit.NewToolCall("", t.def.Name, args))
	if err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", fmt.Errorf("%s", result.Error)
	}

	return result.Output, nil
}
