package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

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
	if text := toolResultText(res); text != "" {
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

func toolResultText(res *mcpsdk.CallToolResult) string {
	parts := make([]string, 0, len(res.Content))
	for _, content := range res.Content {
		text := contentText(content)
		if text == "" {
			continue
		}

		parts = append(parts, text)
	}

	return strings.Join(parts, "\n")
}

func contentText(content mcpsdk.Content) string {
	switch c := content.(type) {
	case *mcpsdk.TextContent:
		return c.Text
	case *mcpsdk.ImageContent:
		return fmt.Sprintf("[image: %s, %d bytes]", c.MIMEType, len(c.Data))
	case *mcpsdk.AudioContent:
		return fmt.Sprintf("[audio: %s, %d bytes]", c.MIMEType, len(c.Data))
	case *mcpsdk.ResourceLink:
		return fmt.Sprintf("[resource: %s]", c.URI)
	case *mcpsdk.EmbeddedResource:
		if c.Resource.Text != "" {
			return c.Resource.Text
		}

		return fmt.Sprintf("[resource: %s, %s, %d bytes]", c.Resource.URI, c.Resource.MIMEType, len(c.Resource.Blob))
	default:
		data, err := json.Marshal(content)
		if err != nil {
			return ""
		}

		return string(data)
	}
}
