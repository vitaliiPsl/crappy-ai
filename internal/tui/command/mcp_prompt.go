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
}

type mcpPromptProvider struct {
	source MCPPromptSource
}

func NewMCPPromptProvider(source MCPPromptSource) Provider {
	if source == nil {
		return nil
	}

	return mcpPromptProvider{source: source}
}

func (p mcpPromptProvider) Commands(ctx context.Context) []Command {
	prompts := p.source.GetMCPPrompts(ctx)

	cmds := make([]Command, 0, len(prompts))
	for _, prompt := range prompts {
		cmds = append(cmds, NewMCPPromptCommand(p.source, prompt))
	}

	return cmds
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

func (c *MCPPromptCommand) Execute(_ context.Context, req Request) tea.Cmd {
	return func() tea.Msg {
		args, err := promptArgs(c.prompt.Arguments, req.Args)
		if err != nil {
			return SystemMsg{Text: err.Error()}
		}

		return SubmitMCPPromptMsg{
			Server: c.prompt.Server,
			Name:   c.prompt.Name,
			Args:   args,
		}
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
