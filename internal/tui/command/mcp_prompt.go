package command

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
)

type MCPPromptSource interface {
	GetMCPPrompts(ctx context.Context) []mcp.ServerPrompt
	GetMCPPrompt(ctx context.Context, server, name string, args map[string]string) (mcp.PromptResult, error)
}

type MCPPromptCommand struct {
	source MCPPromptSource
	prompt mcp.ServerPrompt
}

func NewMCPPromptCommand(source MCPPromptSource, prompt mcp.ServerPrompt) *MCPPromptCommand {
	return &MCPPromptCommand{
		source: source,
		prompt: prompt,
	}
}

func (c *MCPPromptCommand) Definition() Definition {
	return Definition{
		Name:        MCPPromptCommandName(c.prompt),
		Description: c.prompt.Description,
		Kind:        KindMCP,
	}
}

func (c *MCPPromptCommand) Execute(ctx context.Context, req Request) tea.Cmd {
	return func() tea.Msg {
		args, err := promptArgs(c.prompt.Arguments, req.Args)
		if err != nil {
			return SystemMsg{Text: err.Error()}
		}

		result, err := c.source.GetMCPPrompt(ctx, c.prompt.Server, c.prompt.Name, args)
		if err != nil {
			return SystemMsg{Text: fmt.Sprintf("MCP prompt %s failed: %v", MCPPromptCommandName(c.prompt), err)}
		}

		text := FormatPromptResult(result)
		if strings.TrimSpace(text) == "" {
			return SystemMsg{Text: fmt.Sprintf("MCP prompt %s returned no text", MCPPromptCommandName(c.prompt))}
		}

		return SubmitTextMsg{Text: text}
	}
}

func MCPPromptCommandName(prompt mcp.ServerPrompt) string {
	return "mcp:" + prompt.Server + ":" + prompt.Name
}

func promptArgs(defs []mcp.PromptArgument, values []string) (map[string]string, error) {
	args := make(map[string]string, len(defs))
	positional := make([]string, 0, len(values))
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if ok && key != "" {
			args[key] = val

			continue
		}

		positional = append(positional, value)
	}

	pos := 0
	for _, def := range defs {
		if _, exists := args[def.Name]; exists {
			continue
		}

		if pos >= len(positional) {
			if def.Required {
				return nil, fmt.Errorf("missing required argument %q", def.Name)
			}

			continue
		}

		args[def.Name] = positional[pos]
		pos++
	}

	return args, nil
}

func FormatPromptResult(result mcp.PromptResult) string {
	var parts []string
	for _, message := range result.Messages {
		for _, content := range message.Content {
			if text := promptContentText(content); text != "" {
				parts = append(parts, text)
			}
		}
	}

	return strings.Join(parts, "\n\n")
}

func promptContentText(content mcp.PromptContent) string {
	switch content.Type {
	case "text":
		return content.Text
	case "resource":
		if content.Resource != nil && content.Resource.Text != "" {
			return content.Resource.Text
		}

		if content.Text != "" {
			return content.Text
		}

		return fmt.Sprintf("[resource: %s, %s]", content.URI, content.MIMEType)
	case "resource_link":
		return fmt.Sprintf("[resource: %s]", content.URI)
	case "image":
		return fmt.Sprintf("[image: %s, %d bytes]", content.MIMEType, len(content.Data))
	case "audio":
		return fmt.Sprintf("[audio: %s, %d bytes]", content.MIMEType, len(content.Data))
	default:
		return ""
	}
}
