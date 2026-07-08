package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

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
	parts := make([]string, 0, 2)
	if text := kit.ContentsTextFallback(convertToolResultContent(res)); text != "" {
		parts = append(parts, text)
	}

	if res.StructuredContent != nil {
		data, err := json.MarshalIndent(res.StructuredContent, "", "  ")
		if err == nil {
			parts = append(parts, string(data))
		}
	}

	text := strings.Join(parts, "\n")
	if res.IsError {
		if text == "" {
			text = "MCP tool returned an error"
		}

		return kit.NewToolResult(call, text, fmt.Errorf("%s", text))
	}

	return kit.NewToolResult(call, text, nil)
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
