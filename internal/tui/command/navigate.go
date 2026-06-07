package command

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

type NewCommand struct{}

func NewNewCommand() *NewCommand {
	return &NewCommand{}
}

func (c *NewCommand) Definition() Definition {
	return Definition{Name: "new", Description: "Start a new session"}
}

func (c *NewCommand) Execute(_ context.Context, _ Request) tea.Cmd {
	return func() tea.Msg { return NavNewSessionMsg{} }
}

type SessionsCommand struct{}

func NewSessionsCommand() *SessionsCommand {
	return &SessionsCommand{}
}

func (c *SessionsCommand) Definition() Definition {
	return Definition{Name: "sessions", Description: "Open the sessions screen"}
}

func (c *SessionsCommand) Execute(_ context.Context, _ Request) tea.Cmd {
	return func() tea.Msg { return NavSessionsMsg{} }
}

type SettingsCommand struct{}

func NewSettingsCommand() *SettingsCommand {
	return &SettingsCommand{}
}

func (c *SettingsCommand) Definition() Definition {
	return Definition{Name: "settings", Description: "Open the settings screen"}
}

func (c *SettingsCommand) Execute(_ context.Context, _ Request) tea.Cmd {
	return func() tea.Msg { return NavSettingsMsg{} }
}

type MCPCommand struct{}

func NewMCPCommand() *MCPCommand {
	return &MCPCommand{}
}

func (c *MCPCommand) Definition() Definition {
	return Definition{Name: "mcp", Description: "Open the MCP clients screen"}
}

func (c *MCPCommand) Execute(_ context.Context, _ Request) tea.Cmd {
	return func() tea.Msg { return NavMCPMsg{} }
}

type JobsCommand struct{}

func NewJobsCommand() *JobsCommand {
	return &JobsCommand{}
}

func (c *JobsCommand) Definition() Definition {
	return Definition{Name: "jobs", Description: "Open the background jobs screen"}
}

func (c *JobsCommand) Execute(_ context.Context, _ Request) tea.Cmd {
	return func() tea.Msg { return NavJobsMsg{} }
}
