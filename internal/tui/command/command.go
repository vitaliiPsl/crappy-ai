package command

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

type Definition struct {
	Name        string
	Description string
}

type Command interface {
	Definition() Definition
	Execute(ctx context.Context, req Request) tea.Cmd
}

type Request struct {
	SessionID string
	Args      []string
}

type SystemMsg struct {
	Text string
}

type NavNewSessionMsg struct{}

type NavSessionsMsg struct{}

type NavSettingsMsg struct{}
