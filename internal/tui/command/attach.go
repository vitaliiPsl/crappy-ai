package command

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type AttachCommand struct{}

func NewAttachCommand() *AttachCommand {
	return &AttachCommand{}
}

func (c *AttachCommand) Definition() Definition {
	return Definition{Name: "attach", Description: "Attach a file to the next message"}
}

func (c *AttachCommand) Execute(_ context.Context, req Request) tea.Cmd {
	path := strings.TrimSpace(strings.Join(req.Args, " "))
	if path == "" {
		return func() tea.Msg { return SystemMsg{Text: "Usage: /attach <path>"} }
	}

	return func() tea.Msg { return AttachFileMsg{Path: path} }
}
