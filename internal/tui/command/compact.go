package command

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

type CompactSessionMsg struct{}

type CompactCommand struct{}

func NewCompactCommand() *CompactCommand {
	return &CompactCommand{}
}

func (c *CompactCommand) Definition() Definition {
	return Definition{Name: "compact", Description: "Summarize and compact the conversation"}
}

func (c *CompactCommand) Execute(_ context.Context, _ Request) tea.Cmd {
	return func() tea.Msg { return CompactSessionMsg{} }
}
