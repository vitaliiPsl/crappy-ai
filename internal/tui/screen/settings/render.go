package settings

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

const (
	headerText = "Settings"
	hintsText  = "j/Down Move • Enter Edit • Left/Right Cycle • s Save • Esc Back"
	editHints  = "Enter Confirm • Shift+Enter New Line • Esc Cancel"
	pickHints  = "Type Filter • j/Down Move • Enter Select • Esc Cancel"
	editPrefix = "Editing "

	emptyValue     = "(none)"
	savingText     = "Saving..."
	savedText      = "Saved"
	dirtyMarker    = " *"
	cursorPrefix   = "> "
	noCursorPrefix = "  "
	valueSep       = "  "
	errorPrefix    = "Error: "
	maskText       = "********"
	truncatedMark  = "..."
	defaultOption  = "default"
	optionSep      = " / "
	tabWidth       = 16

	headerLines  = 2
	hintsHeight  = 1
	previewLines = 3
)

var (
	thm = theme.Default

	headerStyle   = lipgloss.NewStyle().Foreground(thm.Primary).Bold(true)
	sectionStyle  = lipgloss.NewStyle().Foreground(thm.Secondary).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(thm.Primary).Bold(true)
	labelStyle    = lipgloss.NewStyle().Foreground(thm.Text)
	valueStyle    = lipgloss.NewStyle().Foreground(thm.SubtleText)
	mutedStyle    = lipgloss.NewStyle().Foreground(thm.Muted)
	errorStyle    = lipgloss.NewStyle().Foreground(thm.Error)
	successStyle  = lipgloss.NewStyle().Foreground(thm.Success)
	warningStyle  = lipgloss.NewStyle().Foreground(thm.Warning)
	hintsStyle    = lipgloss.NewStyle().Foreground(thm.SubtleText)
)

func (m Model) View() string {
	title := headerText
	if isDirtyState(m.state) {
		title += warningStyle.Render(dirtyMarker)
	}

	header := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(headerStyle.Render(title))

	hintsView := hintsStyle.Width(m.width).Align(lipgloss.Center).Render(m.hintsText())

	parts := []string{header, "", m.fields.View()}
	if suggestions := m.modelPickerView(); suggestions != "" {
		parts = append(parts, suggestions)
	}

	if input := m.inputView(); input != "" {
		parts = append(parts, input)
	}

	if status := m.statusView(); status != "" {
		parts = append(parts, status)
	}

	parts = append(parts, hintsView)

	return strings.Join(parts, "\n")
}

func (m *Model) refreshContent() {
	m.fields.SetRows(m.fieldRows())
}

func (m Model) fieldRows() []fieldRow {
	rows := make([]fieldRow, 0, len(m.fields.defs))
	for _, field := range m.fields.defs {
		rows = append(rows, fieldRow{
			section: field.section,
			label:   field.label,
			value:   m.renderFieldValue(field),
		})
	}

	return rows
}

func (m Model) hintsText() string {
	switch m.state {
	case stateEditing:
		return editHints
	case statePickingModel:
		return pickHints
	default:
		return hintsText
	}
}

func (m Model) renderFieldValue(field fieldDef) string {
	value := field.get(m)
	if field.masked && value != "" {
		return valueStyle.Render(maskText)
	}

	if field.kind == fieldModel {
		return modelPreview(value, m.hasModel(value))
	}

	if value == "" {
		value = emptyValue
	}

	if field.kind == fieldTextarea {
		value = preview(value, previewLines)
	}

	if field.kind == fieldOption {
		return valueStyle.Render(optionPreview(value, field.options(m)))
	}

	return valueStyle.Render(value)
}

func (m Model) inputView() string {
	if m.state != stateEditing && m.state != statePickingModel {
		return ""
	}

	field := m.fields.currentField()
	label := mutedStyle.Render(editPrefix) + selectedStyle.Render(field.label)

	if m.state == statePickingModel {
		return m.input.View()
	}

	return label + "\n" + m.input.View()
}

func (m Model) modelPickerView() string {
	if m.state != statePickingModel {
		return ""
	}

	return m.modelPicker.View()
}

func (m *Model) resizeViewport() {
	inputHeight := 0
	if input := m.inputView(); input != "" {
		inputHeight = lipgloss.Height(input)
	}

	suggestionsHeight := 0
	if suggestions := m.modelPickerView(); suggestions != "" {
		suggestionsHeight = lipgloss.Height(suggestions)
	}

	statusHeight := 0
	if status := m.statusView(); status != "" {
		statusHeight = lipgloss.Height(status)
	}

	m.fields.SetSize(m.width, max(m.height-headerLines-hintsHeight-inputHeight-suggestionsHeight-statusHeight, 1))
}

func (m Model) statusView() string {
	switch m.state {
	case stateFailed:
		return errorStyle.Width(m.width).Render(errorPrefix + m.saveErr.Error())
	case stateSaving:
		return mutedStyle.Width(m.width).Render(savingText)
	case stateSaved:
		return successStyle.Width(m.width).Render(savedText)
	default:
		return ""
	}
}

func labelCell(label string) string {
	return lipgloss.NewStyle().Width(tabWidth).Render(label)
}

func preview(value string, lines int) string {
	parts := strings.Split(value, "\n")
	if len(parts) > lines {
		parts = append(parts[:lines], truncatedMark)
	}

	return strings.Join(parts, "\n")
}

func optionPreview(value string, options []string) string {
	if value == "" {
		value = defaultOption
	}

	if len(options) == 0 {
		return value
	}

	var parts []string
	for _, option := range options {
		label := option
		if label == "" {
			label = defaultOption
		}

		if option == value || (option == "" && value == defaultOption) {
			parts = append(parts, selectedStyle.Render(label))
		} else {
			parts = append(parts, mutedStyle.Render(label))
		}
	}

	return strings.Join(parts, optionSep)
}

func modelPreview(value string, known bool) string {
	if value == "" {
		return valueStyle.Render(emptyValue)
	}

	if known {
		return valueStyle.Render(value)
	}

	return valueStyle.Render(value) + " " + mutedStyle.Render("(custom)")
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}

	if value > maxValue {
		return maxValue
	}

	return value
}
