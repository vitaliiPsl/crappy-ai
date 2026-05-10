package settings

import (
	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/component"
)

func (m Model) updateEditing(msg tea.Msg) (Model, tea.Cmd) {
	field := m.currentField()

	var (
		cmd tea.Cmd
		out tea.Msg
	)

	m.input, cmd, out = m.input.Update(msg)

	m.resizeViewport()
	m.refreshContent()

	switch out := out.(type) {
	case component.CancelMsg:
		m.state = stateDirty
		m.resizeViewport()
		m.refreshContent()

		return m, nil

	case component.ConfirmMsg:
		field.set(&m, out.Value)
		m.state = stateDirty
		m.saveErr = nil
		m.resizeViewport()
		m.refreshContent()

		return m, nil
	}

	return m, cmd
}

func (m Model) startEditing() (Model, tea.Cmd) {
	field := m.currentField()
	value := field.get(m)

	m.input = component.NewInput(
		component.WithMultiline(field.kind == fieldTextarea),
		component.WithMasked(field.masked),
	)
	m.input.SetWidth(m.width)
	m.input.SetValue(value)
	m.state = stateEditing
	m.resizeViewport()
	m.refreshContent()

	return m, m.input.Focus()
}
