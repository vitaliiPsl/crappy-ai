package memory

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	corememory "github.com/vitaliiPsl/crappy-ai/internal/memory"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/component"
)

const (
	headerText          = "Memories"
	hintsText           = "j/k/Up/Down Move • Enter Edit • n New • d Delete • r Refresh • Esc Back"
	editHintsText       = "Tab Kind • Enter Save • Shift+Enter New Line • Esc Cancel"
	emptyTitle          = "No memories saved"
	emptySubtitle       = "Press n to add a memory."
	deleteConfirmPrompt = "Forget memory?"
	errorPrefix         = "Error: "
	inputPlaceholder    = "One concise, durable memory"
	cursorPrefix        = "> "
	noCursorPrefix      = "  "
	metaPad             = "  "
	metaSep             = " · "
	timestampFormat     = "Jan 02 15:04"
	headerLines         = 2
	itemHeight          = 3
	previewLength       = 180
)

var memoryKinds = []corememory.Kind{
	corememory.KindProfile,
	corememory.KindPreference,
	corememory.KindInstruction,
}

type state int

const (
	stateBrowsing state = iota
	stateEditing
	stateDeleting
)

type Model struct {
	ctx    context.Context
	server *server.Server

	memories []corememory.Memory
	cursor   int
	err      error
	state    state

	draft         corememory.Memory
	input         component.Input
	deleteConfirm component.Confirm

	viewport viewport.Model
	width    int
	height   int
}

func New(ctx context.Context, srv *server.Server) Model {
	vp := viewport.New()
	vp.SoftWrap = false
	input := component.NewInput(
		component.WithMultiline(true),
		component.WithPlaceholder(inputPlaceholder),
	)

	return Model{
		ctx:      ctx,
		server:   srv,
		input:    input,
		viewport: vp,
		deleteConfirm: component.NewConfirm(
			component.WithConfirmPrompt(deleteConfirmPrompt),
			component.WithCancelKeys("n", "esc", "d"),
		),
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadMemories("")
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case memoriesLoadedMsg:
		m.err = msg.err
		m.memories = msg.memories

		m.state = stateBrowsing
		if msg.editing {
			m.state = stateEditing
		}

		m.selectMemory(msg.selectID)
		m.resizeViewport()
		m.refreshContent()
		m.scrollToCursor()

		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case stateEditing:
			return m.updateEditing(msg)
		case stateDeleting:
			return m.updateDeleting(msg)
		default:
			return m.updateBrowsing(msg)
		}
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	header := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(headerStyle.Render(headerText))
	bottom := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(m.bottomView())

	parts := []string{header, "", m.viewport.View()}
	if m.state == stateEditing {
		parts = append(parts, m.editView())
	}

	if m.err != nil {
		parts = append(parts, errorStyle.Render(errorPrefix+m.err.Error()))
	}

	parts = append(parts, bottom)

	return strings.Join(parts, "\n")
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.SetWidth(width)
	m.input.SetWidth(width)
	m.resizeViewport()
	m.refreshContent()
}

func (m Model) updateBrowsing(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.String() {
	case "up", "k":
		return m.moveCursor(-1)
	case "down", "j":
		return m.moveCursor(1)
	case "n":
		return m.startCreating()
	case "e", "enter":
		return m.startEditing()
	case "d":
		return m.requestDelete()
	case "r":
		return m, m.loadMemories(m.selectedID())
	case "esc":
		return m, func() tea.Msg { return ClosedMsg{} }
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(key)

	return m, cmd
}

func (m Model) updateEditing(msg tea.Msg) (Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab":
			m.cycleKind()

			return m, nil
		}
	}

	var (
		cmd tea.Cmd
		out tea.Msg
	)

	m.input, cmd, out = m.input.Update(msg)

	switch out := out.(type) {
	case component.CancelMsg:
		m.state = stateBrowsing
		m.err = nil
		m.resizeViewport()

		return m, nil
	case component.ConfirmMsg:
		m.draft.Content = out.Value
		m.state = stateBrowsing
		m.resizeViewport()

		return m, m.saveDraft()
	}

	m.resizeViewport()

	return m, cmd
}

func (m Model) updateDeleting(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd tea.Cmd
		out tea.Msg
	)

	m.deleteConfirm, cmd, out = m.deleteConfirm.Update(msg)

	switch out.(type) {
	case component.ConfirmMsg:
		id := m.selectedID()
		m.state = stateBrowsing
		m.resizeViewport()

		return m, m.deleteMemory(id)
	case component.CancelMsg:
		m.state = stateBrowsing
		m.resizeViewport()

		return m, nil
	}

	return m, cmd
}

func (m Model) startCreating() (Model, tea.Cmd) {
	m.draft = corememory.Memory{Kind: corememory.KindProfile}

	return m.beginEditing()
}

func (m Model) startEditing() (Model, tea.Cmd) {
	if len(m.memories) == 0 {
		return m, nil
	}

	m.draft = m.memories[m.cursor]

	return m.beginEditing()
}

func (m Model) beginEditing() (Model, tea.Cmd) {
	m.state = stateEditing
	m.err = nil
	m.input = component.NewInput(
		component.WithMultiline(true),
		component.WithPlaceholder(inputPlaceholder),
	)
	m.input.SetWidth(m.width)
	m.input.SetValue(m.draft.Content)
	m.resizeViewport()

	return m, m.input.Focus()
}

func (m Model) requestDelete() (Model, tea.Cmd) {
	if len(m.memories) == 0 {
		return m, nil
	}

	m.state = stateDeleting
	m.resizeViewport()

	return m, nil
}

func (m Model) moveCursor(delta int) (Model, tea.Cmd) {
	next := m.cursor + delta
	if next < 0 || next >= len(m.memories) {
		return m, nil
	}

	m.cursor = next
	m.refreshContent()
	m.scrollToCursor()

	return m, nil
}

func (m *Model) cycleKind() {
	index := 0
	for i, kind := range memoryKinds {
		if kind == m.draft.Kind {
			index = i

			break
		}
	}

	index = (index + 1) % len(memoryKinds)
	m.draft.Kind = memoryKinds[index]
}

func (m Model) saveDraft() tea.Cmd {
	draft := m.draft

	return func() tea.Msg {
		var (
			saved corememory.Memory
			err   error
		)

		if draft.ID == "" {
			saved, err = m.server.CreateMemory(m.ctx, corememory.CreateParams{
				Kind: draft.Kind, Content: draft.Content,
			})
		} else {
			saved, err = m.server.UpdateMemory(m.ctx, corememory.UpdateParams{
				ID: draft.ID, Kind: draft.Kind, Content: draft.Content,
			})
		}

		if err != nil {
			return memoriesLoadedMsg{memories: m.memories, selectID: draft.ID, editing: true, err: err}
		}

		memories, err := m.server.ListMemories(m.ctx)

		return memoriesLoadedMsg{memories: memories, selectID: saved.ID, err: err}
	}
}

func (m Model) deleteMemory(id string) tea.Cmd {
	return func() tea.Msg {
		deleteErr := m.server.DeleteMemory(m.ctx, id)

		memories, err := m.server.ListMemories(m.ctx)
		if err == nil && deleteErr != nil {
			err = deleteErr
		}

		return memoriesLoadedMsg{memories: memories, err: err}
	}
}

func (m Model) loadMemories(selectID string) tea.Cmd {
	return func() tea.Msg {
		memories, err := m.server.ListMemories(m.ctx)

		return memoriesLoadedMsg{memories: memories, selectID: selectID, err: err}
	}
}

func (m *Model) selectMemory(id string) {
	if len(m.memories) == 0 {
		m.cursor = 0

		return
	}

	if id != "" {
		for i, item := range m.memories {
			if item.ID == id {
				m.cursor = i

				return
			}
		}
	}

	m.cursor = min(m.cursor, len(m.memories)-1)
}

func (m Model) selectedID() string {
	if m.cursor < 0 || m.cursor >= len(m.memories) {
		return ""
	}

	return m.memories[m.cursor].ID
}

func (m Model) bottomView() string {
	switch m.state {
	case stateEditing:
		return hintsStyle.Render(editHintsText)
	case stateDeleting:
		return m.deleteConfirm.View()
	default:
		return hintsStyle.Render(hintsText)
	}
}

func (m Model) editView() string {
	return metaStyle.Render("Kind: ") + kindBadge(m.draft.Kind) + "\n" + m.input.View()
}

func (m *Model) resizeViewport() {
	reserved := headerLines + lipgloss.Height(m.bottomView())
	if m.state == stateEditing {
		reserved += lipgloss.Height(m.editView())
	}

	if m.err != nil {
		reserved++
	}

	m.viewport.SetHeight(max(m.height-reserved, 1))
}

func (m *Model) refreshContent() {
	switch {
	case len(m.memories) == 0:
		m.viewport.SetContent(renderEmpty(m.width, m.viewport.Height()))
	default:
		m.viewport.SetContent(renderList(m.memories, m.cursor))
	}
}

func (m *Model) scrollToCursor() {
	itemStart := m.cursor * itemHeight
	itemEnd := itemStart + itemHeight - 1
	height := m.viewport.Height()
	offset := m.viewport.YOffset()

	switch {
	case itemStart < offset:
		m.viewport.SetYOffset(itemStart)
	case itemEnd >= offset+height:
		m.viewport.SetYOffset(max(itemEnd-height+1, 0))
	}
}

func renderEmpty(width, height int) string {
	content := emptyStyle.Render(emptyTitle) + "\n" + emptyStyle.Render(emptySubtitle)

	return lipgloss.NewStyle().
		Width(width).
		Height(max(height, 1)).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(content)
}

func renderList(memories []corememory.Memory, cursor int) string {
	var b strings.Builder
	for i, item := range memories {
		b.WriteString(renderMemory(item, i == cursor))
		b.WriteString("\n\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderMemory(item corememory.Memory, selected bool) string {
	cursor := noCursorPrefix

	content := itemStyle.Render(memoryPreview(item.Content))
	if selected {
		cursor = selectedStyle.Render(cursorPrefix)
		content = selectedStyle.Render(memoryPreview(item.Content))
	}

	meta := []string{item.UpdatedAt.Format(timestampFormat)}
	if item.UpdatedAt != item.CreatedAt {
		meta = append(meta, "updated")
	}

	return cursor + content + "\n" + metaPad + kindBadge(item.Kind) + metaStyle.Render(metaSep+strings.Join(meta, metaSep))
}

func memoryPreview(content string) string {
	content = strings.Join(strings.Fields(content), " ")

	runes := []rune(content)
	if len(runes) > previewLength {
		return string(runes[:previewLength]) + "..."
	}

	return content
}

func kindBadge(kind corememory.Kind) string {
	switch kind {
	case corememory.KindProfile:
		return profileStyle.Render(string(kind))
	case corememory.KindPreference:
		return preferenceStyle.Render(string(kind))
	case corememory.KindInstruction:
		return instructionStyle.Render(string(kind))
	default:
		return string(kind)
	}
}
