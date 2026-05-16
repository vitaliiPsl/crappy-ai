package session

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
)

type footer struct {
	turn   turnIndicator
	input  inputBar
	status statusBar
}

func newFooter(registry *command.Registry, model, cwd string) footer {
	return footer{
		turn:   newTurnIndicator(),
		input:  newInputBar(registry),
		status: newStatusBar(model, cwd),
	}
}

func (f footer) Init() tea.Cmd {
	return f.input.Init()
}

func (f footer) Update(msg tea.Msg) (footer, tea.Cmd, bool) {
	var cmds []tea.Cmd

	var (
		cmd      tea.Cmd
		consumed bool
	)

	f.turn, cmd, consumed = f.turn.Update(msg)
	cmds = append(cmds, cmd)

	f.status, cmd = f.status.Update(msg)
	cmds = append(cmds, cmd)

	if consumed {
		return f, tea.Batch(cmds...), true
	}

	if f.turn.Active() {
		return f, tea.Batch(cmds...), false
	}

	f.input, cmd, consumed = f.input.Update(msg)
	cmds = append(cmds, cmd)

	return f, tea.Batch(cmds...), consumed
}

func (f footer) View() string {
	var parts []string

	if turn := f.turn.View(); turn != "" {
		parts = append(parts, turn)
	}

	parts = append(parts, f.input.View())

	if status := f.status.View(); status != "" {
		parts = append(parts, status)
	}

	return strings.Join(parts, "\n")
}

func (f footer) Height() int {
	return lipgloss.Height(f.View())
}

func (f *footer) setSize(width int) {
	f.turn.setSize(width)
	f.input.setSize(width)
	f.status.setSize(width)
}
