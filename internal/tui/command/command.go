package command

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

type Kind int

const (
	KindBuiltin Kind = iota
	KindSkill
)

type Definition struct {
	Name        string
	Description string
	Kind        Kind
}

type Command interface {
	Definition() Definition
	Execute(ctx context.Context, req Request) tea.Cmd
}

type Request struct {
	SessionID string
	Args      []string
	Raw       string
}

type SystemMsg struct {
	Text string
}

type SubmitTextMsg struct {
	Text string
}

type SubmitSkillMsg struct {
	Text string
	Name string
	Args []string
}

type NavNewSessionMsg struct{}

type NavSessionsMsg struct{}

type NavSettingsMsg struct{}

type NavMCPMsg struct{}
