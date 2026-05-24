package settings

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/component"
)

func (m Model) updateEditing(msg tea.Msg) (Model, tea.Cmd) {
	field := m.fields.currentField()

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

func (m Model) startEditing(field fieldDef) (Model, tea.Cmd) {
	m.input = component.NewInput(
		component.WithMultiline(field.kind == fieldTextarea),
		component.WithMasked(field.masked),
	)
	m.input.SetWidth(m.width)
	m.input.SetValue(field.get(m))
	m.state = stateEditing
	m.resizeViewport()
	m.refreshContent()

	return m, m.input.Focus()
}

func (m Model) updatePickingModel(msg tea.Msg) (Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k":
			m.modelPicker.Previous()
			m.resizeViewport()
			m.refreshContent()

			return m, nil

		case "down", "j":
			m.modelPicker.Next()
			m.resizeViewport()
			m.refreshContent()

			return m, nil

		case "esc":
			m.state = m.returnState
			m.resizeViewport()
			m.refreshContent()

			return m, nil

		case "enter":
			if modelID, ok := m.pickModelID(m.input.Value()); ok {
				m.cfg.Model = modelID
				m.state = stateDirty
				m.saveErr = nil
			}

			m.resizeViewport()
			m.refreshContent()

			return m, nil
		}
	}

	var (
		cmd tea.Cmd
		out tea.Msg
	)

	m.input, cmd, out = m.input.Update(msg)
	m.modelPicker.Update(m.input.Value())

	switch out.(type) {
	case component.CancelMsg:
		m.state = m.returnState
	case component.ConfirmMsg:
		if modelID, ok := m.pickModelID(m.input.Value()); ok {
			m.cfg.Model = modelID
			m.state = stateDirty
			m.saveErr = nil
		}
	}

	m.resizeViewport()
	m.refreshContent()

	return m, cmd
}

func (m Model) startModelPicking() (Model, tea.Cmd) {
	models := m.modelOptions()
	if len(models) == 0 {
		return m.startEditing(m.fields.currentField())
	}

	m.input = component.NewInput(
		component.WithPlaceholder(modelPickerPlaceholder),
		component.WithPrompt(modelPickerPrompt),
	)
	m.input.SetWidth(m.width)
	m.modelPicker.SetModels(models, m.cfg.Model)
	m.returnState = m.state
	m.state = statePickingModel
	m.resizeViewport()
	m.refreshContent()

	return m, m.input.Focus()
}

func (m Model) pickModelID(input string) (string, bool) {
	if model, ok := m.modelPicker.Selected(); ok {
		return model.ID, true
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return "", false
	}

	return input, true
}
