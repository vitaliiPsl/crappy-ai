package session

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
)

type footer struct {
	input  inputBar
	status statusBar
}

func newFooter(registry *command.Registry, model, cwd string) footer {
	return footer{
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

	f.status, cmd, consumed = f.status.Update(msg)

	cmds = append(cmds, cmd)
	if consumed {
		return f, tea.Batch(cmds...), true
	}

	if f.status.TurnActive() {
		return f, tea.Batch(cmds...), false
	}

	f.input, cmd, consumed = f.input.Update(msg)
	cmds = append(cmds, cmd)

	return f, tea.Batch(cmds...), consumed
}

func (f footer) View() string {
	var parts []string

	if status := f.status.StatusView(); status != "" {
		parts = append(parts, status)
	}

	if suggestions := f.input.SuggestionsView(); suggestions != "" {
		parts = append(parts, suggestions)
	}

	parts = append(parts, strings.TrimRight(f.input.View(), "\n"))

	if meta := f.status.MetaView(); meta != "" {
		parts = append(parts, meta)
	}

	if hints := f.status.HintsView(); hints != "" {
		parts = append(parts, hints)
	}

	return strings.Join(parts, "\n")
}

func (f footer) Height() int {
	return lipgloss.Height(f.View())
}

func (f *footer) setSize(width int) {
	f.input.setSize(width)
	f.status.setSize(width)
}
