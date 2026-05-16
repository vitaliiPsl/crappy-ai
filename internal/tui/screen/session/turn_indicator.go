package session

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

const defaultTurnLabel = "Generating..."

type turnIndicator struct {
	spinner spinner.Model
	active  bool
	width   int
}

func newTurnIndicator() turnIndicator {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(sessionTheme.Primary)

	return turnIndicator{spinner: sp}
}

func (t turnIndicator) Update(msg tea.Msg) (turnIndicator, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case sessionEventMsg:
		switch msg.event.Type {
		case sessiondata.EventTurnComplete,
			sessiondata.EventTurnCancelled,
			sessiondata.EventError:
			t.active = false
		}

		return t, nil, false

	case turnStartedMsg:
		t.active = true

		return t, t.spinner.Tick, false

	case turnStoppedMsg:
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

func (t turnIndicator) View() string {
	if !t.active {
		return ""
	}

	return subtleTextStyle.Width(t.width).Render(t.spinner.View() + " " + defaultTurnLabel)
}

func (t turnIndicator) Active() bool {
	return t.active
}

func (t *turnIndicator) setSize(width int) {
	t.width = width
}
