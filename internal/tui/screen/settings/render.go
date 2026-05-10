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

	hints := hintsText
	if m.state == stateEditing {
		hints = editHints
	}

	hintsView := hintsStyle.Width(m.width).Align(lipgloss.Center).Render(hints)

	parts := []string{header, "", m.viewport.View()}
	if editor := m.editorView(); editor != "" {
		parts = append(parts, editor)
	}

	if status := m.statusView(); status != "" {
		parts = append(parts, status)
	}

	parts = append(parts, hintsView)

	return strings.Join(parts, "\n")
}

func (m *Model) refreshContent() {
	if len(m.fields) == 0 {
		m.viewport.SetContent("")

		return
	}

	var b strings.Builder

	currentSection := ""
	for i, field := range m.fields {
		if field.section != currentSection {
			if currentSection != "" {
				b.WriteByte('\n')
			}

			currentSection = field.section
			b.WriteString(sectionStyle.Render(currentSection))
			b.WriteByte('\n')
		}

		cursor := noCursorPrefix

		style := labelStyle
		if i == m.cursor {
			cursor = cursorPrefix
			style = selectedStyle
		}

		b.WriteString(cursor)
		b.WriteString(style.Render(labelCell(field.label)))
		b.WriteString(valueSep)
		b.WriteString(m.renderFieldValue(field))
		b.WriteByte('\n')
	}

	m.viewport.SetContent(strings.TrimRight(b.String(), "\n"))
}

func (m Model) renderFieldValue(field fieldDef) string {
	value := field.get(m)
	if field.masked && value != "" {
		return valueStyle.Render(maskText)
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

func (m Model) editorView() string {
	if m.state != stateEditing {
		return ""
	}

	field := m.currentField()
	label := mutedStyle.Render(editPrefix) + selectedStyle.Render(field.label)

	return label + "\n" + m.editor.View()
}

func (m *Model) resizeViewport() {
	editorHeight := 0
	if editor := m.editorView(); editor != "" {
		editorHeight = lipgloss.Height(editor)
	}

	statusHeight := 0
	if status := m.statusView(); status != "" {
		statusHeight = lipgloss.Height(status)
	}

	m.viewport.SetHeight(max(m.height-headerLines-hintsHeight-editorHeight-statusHeight, 1))
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

func (m *Model) scrollToCursor() {
	line := m.cursor
	for i := 0; i <= m.cursor && i < len(m.fields); i++ {
		if i == 0 || m.fields[i].section != m.fields[i-1].section {
			line++
		}
	}

	height := m.viewport.Height()

	offset := m.viewport.YOffset()
	if line < offset {
		m.viewport.SetYOffset(line)
	} else if line >= offset+height {
		m.viewport.SetYOffset(line - height + 1)
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

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}

	if value > maxValue {
		return maxValue
	}

	return value
}
