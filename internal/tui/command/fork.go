package command

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

type ForkSessionMsg struct{}

type ForkCommand struct{}

func NewForkCommand() *ForkCommand {
	return &ForkCommand{}
}

func (c *ForkCommand) Definition() Definition {
	return Definition{Name: "fork", Description: "Fork the current conversation"}
}

func (c *ForkCommand) Execute(_ context.Context, _ Request) tea.Cmd {
	return func() tea.Msg { return ForkSessionMsg{} }
}
