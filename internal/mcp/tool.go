package mcp

import (
	"encoding/json"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

var _ kit.Tool = (*tool)(nil)

type tool struct {
	name   string
	def    kit.ToolDefinition
	client Client
}

func newTool(clientName string, client Client, def kit.ToolDefinition) kit.Tool {
	return &tool{
		name:   fmt.Sprintf("mcp__%s__%s", clientName, def.Name),
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

func (t *tool) Execute(rc *kit.RunContext, call kit.ToolCall) (kit.ToolOutput, error) {
	mcpCall := call
	mcpCall.Name = t.def.Name

	result, err := t.client.CallTool(rc.Context, mcpCall)
	if err != nil {
		return kit.ToolOutput{}, err
	}

	if result.Error != "" {
		return kit.ToolOutput{}, fmt.Errorf("%s", result.Error)
	}

	return kit.ToolOutput{
		Content:    result.Output.Content,
		Structured: result.Output.Structured,
	}, nil
}

func convertTools(tools []*mcpsdk.Tool) ([]kit.ToolDefinition, error) {
	defs := make([]kit.ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		schema, err := schemaToMap(tool.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("mcp: tool %q schema: %w", tool.Name, err)
		}

		defs = append(defs, kit.ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			Schema:      schema,
		})
	}

	return defs, nil
}

func schemaToMap(schema any) (map[string]any, error) {
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func convertToolResult(call kit.ToolCall, res *mcpsdk.CallToolResult) kit.ToolResult {
	output := kit.ToolOutput{
		Content:    convertToolResultContent(res),
		Structured: res.StructuredContent,
	}

	if res.IsError {
		text := kit.ContentsText(output.Content)
		if text == "" {
			text = "MCP tool returned an error"
			output.Content = []kit.Content{kit.NewTextContent(text)}
		}

		return kit.NewToolResult(call, output, fmt.Errorf("%s", text))
	}

	return kit.NewToolResult(call, output, nil)
}

func convertToolResultContent(res *mcpsdk.CallToolResult) []kit.Content {
	if res == nil {
		return nil
	}

	out := make([]kit.Content, 0, len(res.Content))
	for _, content := range res.Content {
		item := convertMCPContent(content)
		if item.Type == "" {
			continue
		}

		out = append(out, item)
	}

	return out
}
