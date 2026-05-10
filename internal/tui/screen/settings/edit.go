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

	m.editor, cmd, out = m.editor.Update(msg)

	m.resizeViewport()
	m.refreshContent()

	switch out := out.(type) {
	case component.EditorCancelMsg:
		m.state = stateDirty
		m.resizeViewport()
		m.refreshContent()

		return m, nil

	case component.EditorConfirmMsg:
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

	m.editor = component.NewEditor(
		component.WithEditorMultiline(field.kind == fieldTextarea),
		component.WithEditorMasked(field.masked),
	)
	m.editor.SetWidth(m.width)
	m.editor.SetValue(value)
	m.state = stateEditing
	m.resizeViewport()
	m.refreshContent()

	return m, m.editor.Focus()
}
