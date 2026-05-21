package session

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"

	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

type footer struct {
	run    runIndicator
	input  inputBar
	status statusBar
	prompt *permissionPrompt

	width int
}

func newFooter(registry *command.Registry, model, cwd string) footer {
	return footer{
		run:    newRunIndicator(),
		input:  newInputBar(registry),
		status: newStatusBar(model, cwd),
	}
}

func (f footer) Init() tea.Cmd {
	return f.input.Init()
}

func (f footer) Update(msg tea.Msg) (footer, tea.Cmd, bool) {
	if ev, ok := msg.(sessionEventMsg); ok {
		f.handleEvent(ev.event)
	}

	var cmds []tea.Cmd

	var (
		cmd      tea.Cmd
		consumed bool
	)

	f.run, cmd, consumed = f.run.Update(msg)
	cmds = append(cmds, cmd)

	f.status, cmd = f.status.Update(msg)
	cmds = append(cmds, cmd)

	if consumed {
		return f, tea.Batch(cmds...), true
	}

	if f.prompt != nil {
		if _, ok := msg.(tea.KeyMsg); !ok {
			return f, tea.Batch(cmds...), false
		}

		var (
			promptCmd tea.Cmd
			out       tea.Msg
		)

		*f.prompt, promptCmd, out = f.prompt.Update(msg)
		cmds = append(cmds, promptCmd)

		if pm, ok := out.(permissionPromptMsg); ok {
			f.prompt = nil
			f.status.SetHints("")

			cmds = append(cmds, func() tea.Msg { return pm })
		}

		return f, tea.Batch(cmds...), true
	}

	if f.run.Active() {
		return f, tea.Batch(cmds...), false
	}

	f.input, cmd, consumed = f.input.Update(msg)
	cmds = append(cmds, cmd)

	return f, tea.Batch(cmds...), consumed
}

func (f footer) View() string {
	var parts []string

	if run := f.run.View(); run != "" {
		parts = append(parts, run)
	}

	parts = append(parts, f.bodyView())

	if status := f.status.View(); status != "" {
		parts = append(parts, status)
	}

	return strings.Join(parts, "\n")
}

func (f footer) Height() int {
	return lipgloss.Height(f.View())
}

func (f footer) HasPrompt() bool {
	return f.prompt != nil
}

func (f *footer) setSize(width int) {
	f.width = width
	f.run.setSize(width)
	f.input.setSize(width)
	f.status.setSize(width)

	if f.prompt != nil {
		f.prompt.SetWidth(width)
	}
}

func (f footer) bodyView() string {
	if f.prompt != nil {
		return f.prompt.View()
	}

	return f.input.View()
}

func (f *footer) handleEvent(ev sessiondata.Event) {
	switch ev.Type {
	case sessiondata.EventPermissionPrompt:
		if ev.Prompt == nil {
			return
		}

		p := newPermissionPrompt(ev.Prompt.Request)
		p.SetWidth(f.width)
		f.prompt = &p
		f.status.SetHints(p.HintsText())

	case sessiondata.EventTurnComplete, sessiondata.EventTurnCancelled, sessiondata.EventError:
		f.prompt = nil
		f.status.SetHints("")
	}
}
