package command

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type HelpCommand struct {
	registry *Registry
}

func NewHelpCommand(registry *Registry) *HelpCommand {
	return &HelpCommand{registry: registry}
}

func (c *HelpCommand) Definition() Definition {
	return Definition{Name: "help", Description: "Show available commands"}
}

func (c *HelpCommand) Execute(_ context.Context, _ Request) tea.Cmd {
	return func() tea.Msg {
		var b strings.Builder
		b.WriteString("Available commands:\n")

		for _, def := range c.registry.Definitions() {
			fmt.Fprintf(&b, "  %-12s %s\n", "/"+def.Name, def.Description)
		}

		return SystemMsg{Text: strings.TrimRight(b.String(), "\n")}
	}
}
