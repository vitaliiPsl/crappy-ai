package session

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

const defaultRunLabel = "Generating..."

type runIndicator struct {
	spinner spinner.Model
	active  bool
	width   int
}

func newRunIndicator() runIndicator {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(sessionTheme.Primary)

	return runIndicator{spinner: sp}
}

func (t runIndicator) Update(msg tea.Msg) (runIndicator, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case sessionEventMsg:
		switch msg.event.Type {
		case sessiondata.EventTurnComplete,
			sessiondata.EventTurnCancelled,
			sessiondata.EventError:
			t.active = false
		}

		return t, nil, false

	case runStartedMsg:
		t.active = true

		return t, t.spinner.Tick, false

	case runStoppedMsg:
		t.active = false

		return t, nil, false

	case spinner.TickMsg:
		if !t.active {
			return t, nil, true
		}

		var cmd tea.Cmd

		t.spinner, cmd = t.spinner.Update(msg)

		return t, cmd, true
	}

	return t, nil, false
}

func (t runIndicator) View() string {
	if !t.active {
		return ""
	}

	return subtleTextStyle.Width(t.width).Render(t.spinner.View() + " " + defaultRunLabel)
}

func (t runIndicator) Active() bool {
	return t.active
}

func (t *runIndicator) setSize(width int) {
	t.width = width
}
