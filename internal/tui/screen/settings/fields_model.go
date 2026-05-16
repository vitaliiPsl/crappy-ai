package settings

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type fieldActionKind int

const (
	fieldActionNone fieldActionKind = iota
	fieldActionEdit
	fieldActionPickModel
	fieldActionCycle
)

type fieldAction struct {
	kind  fieldActionKind
	field fieldDef
	delta int
}

type fieldRow struct {
	section string
	label   string
	value   string
}

type fieldsModel struct {
	defs     []fieldDef
	rows     []fieldRow
	cursor   int
	viewport viewport.Model
}

func newFieldsModel(defs []fieldDef) fieldsModel {
	vp := viewport.New()
	vp.SoftWrap = true

	return fieldsModel{
		defs:     defs,
		viewport: vp,
	}
}

func (m fieldsModel) Update(msg tea.Msg) (fieldsModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.refreshContent()
				m.scrollToCursor()
			}

			return m, nil

		case "down", "j":
			if m.cursor < len(m.defs)-1 {
				m.cursor++
				m.refreshContent()
				m.scrollToCursor()
			}

			return m, nil

		case "left":
			field := m.currentField()
			if field.kind == fieldOption {
				return m, fieldActionCmd(fieldAction{kind: fieldActionCycle, field: field, delta: -1})
			}

			return m, nil

		case "right":
			field := m.currentField()
			if field.kind == fieldOption {
				return m, fieldActionCmd(fieldAction{kind: fieldActionCycle, field: field, delta: 1})
			}

			return m, nil

		case "enter":
			field := m.currentField()
			switch field.kind {
			case fieldOption:
				return m, fieldActionCmd(fieldAction{kind: fieldActionCycle, field: field, delta: 1})
			case fieldModel:
				return m, fieldActionCmd(fieldAction{kind: fieldActionPickModel, field: field})
			default:
				return m, fieldActionCmd(fieldAction{kind: fieldActionEdit, field: field})
			}
		}
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func fieldActionCmd(action fieldAction) tea.Cmd {
	return func() tea.Msg { return action }
}

func (m fieldsModel) View() string {
	return m.viewport.View()
}

func (m *fieldsModel) SetRows(rows []fieldRow) {
	m.rows = rows
	m.refreshContent()
	m.scrollToCursor()
}

func (m *fieldsModel) SetSize(width, height int) {
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(max(height, 1))
	m.refreshContent()
	m.scrollToCursor()
}

func (m fieldsModel) currentField() fieldDef {
	if len(m.defs) == 0 {
		return fieldDef{}
	}

	return m.defs[clamp(m.cursor, 0, len(m.defs)-1)]
}

func (m *fieldsModel) refreshContent() {
	if len(m.rows) == 0 {
		m.viewport.SetContent("")

		return
	}

	var b strings.Builder

	currentSection := ""

	for i, row := range m.rows {
		if row.section != currentSection {
			if currentSection != "" {
				b.WriteByte('\n')
			}

			currentSection = row.section
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
		b.WriteString(style.Render(labelCell(row.label)))
		b.WriteString(valueSep)
		b.WriteString(row.value)
		b.WriteByte('\n')
	}

	m.viewport.SetContent(strings.TrimRight(b.String(), "\n"))
}

func (m *fieldsModel) scrollToCursor() {
	line := m.cursor
	for i := 0; i <= m.cursor && i < len(m.rows); i++ {
		if i == 0 || m.rows[i].section != m.rows[i-1].section {
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
