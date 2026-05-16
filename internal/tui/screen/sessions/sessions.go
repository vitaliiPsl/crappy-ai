package sessions

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/server"
	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/component"
	"github.com/vitaliiPsl/crappy-ai/internal/utils"
)

const (
	headerText          = "Sessions"
	hintsText           = "j/Down Move • Enter Open • n New • d Delete • r Refresh • Esc Back"
	emptyTitle          = "No sessions yet"
	emptySubtitle       = "Press n to start a new conversation."
	untitledText        = "Untitled session"
	deleteConfirmPrompt = "Delete session?"
	errorPrefix         = "Error: "
)

const (
	cursorPrefix    = "> "
	noCursorPrefix  = "  "
	titleSep        = "  "
	metaPad         = "  "
	metaSep         = " · "
	timestampFormat = "Jan 02 15:04"
)

const (
	shortIDLen  = 8
	itemHeight  = 3
	headerLines = 2
)

type Model struct {
	ctx    context.Context
	server *server.Server

	sessions []*sessiondata.Session
	cursor   int
	err      error

	pendingDelete bool
	deleteConfirm component.Confirm

	viewport viewport.Model
	width    int
	height   int
}

func New(ctx context.Context, srv *server.Server) Model {
	vp := viewport.New()
	vp.SoftWrap = false

	return Model{
		ctx:      ctx,
		server:   srv,
		viewport: vp,
		deleteConfirm: component.NewConfirm(
			component.WithConfirmPrompt(deleteConfirmPrompt),
			component.WithCancelKeys("n", "esc", "d"),
		),
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadSessions()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sessionsLoadedMsg:
		m.err = msg.err
		m.sessions = msg.sessions
		m.cursor = clampCursor(m.cursor, len(m.sessions))
		m.refreshContent()
		m.scrollToCursor()

		return m, nil

	case tea.KeyMsg:
		if m.pendingDelete {
			return m.handleDelete(msg)
		}

		return m.handleKey(msg)
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	header := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(headerStyle.Render(headerText))
	bottom := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(m.bottomView())

	return header + "\n\n" + m.viewport.View() + "\n" + bottom
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.SetWidth(width)
	m.resizeViewport()
	m.refreshContent()
}

func (m Model) handleKey(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.String() {
	case "up", "k":
		return m.moveCursor(-1)

	case "down", "j":
		return m.moveCursor(1)

	case "enter":
		return m.openSelected()

	case "n":
		return m, func() tea.Msg { return OpenDraftSessionMsg{} }

	case "d":
		return m.requestDelete()

	case "r":
		return m, m.loadSessions()

	case "esc":
		return m, func() tea.Msg { return ClosedMsg{} }
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(key)

	return m, cmd
}

func (m Model) handleDelete(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd tea.Cmd
		out tea.Msg
	)

	m.deleteConfirm, cmd, out = m.deleteConfirm.Update(msg)

	switch out.(type) {
	case component.ConfirmMsg:
		m.pendingDelete = false
		m.resizeViewport()

		return m, m.deleteSession(m.sessions[m.cursor].ID)

	case component.CancelMsg:
		m.pendingDelete = false
		m.resizeViewport()

		return m, nil
	}

	return m, cmd
}

func (m Model) moveCursor(delta int) (Model, tea.Cmd) {
	next := m.cursor + delta
	if next < 0 || next >= len(m.sessions) {
		return m, nil
	}

	m.cursor = next
	m.refreshContent()
	m.scrollToCursor()

	return m, nil
}

func (m Model) openSelected() (Model, tea.Cmd) {
	if len(m.sessions) == 0 {
		return m, nil
	}

	id := m.sessions[m.cursor].ID

	return m, func() tea.Msg { return OpenSessionMsg{SessionID: id} }
}

func (m Model) requestDelete() (Model, tea.Cmd) {
	if len(m.sessions) == 0 {
		return m, nil
	}

	m.pendingDelete = true
	m.resizeViewport()

	return m, nil
}

func (m Model) bottomView() string {
	if m.pendingDelete {
		return m.deleteConfirm.View()
	}

	return hintsStyle.Render(hintsText)
}

func (m *Model) resizeViewport() {
	bottomHeight := lipgloss.Height(m.bottomView())
	m.viewport.SetHeight(max(m.height-headerLines-bottomHeight, 1))
}

func (m *Model) refreshContent() {
	switch {
	case m.err != nil:
		m.viewport.SetContent(errorStyle.Render(errorPrefix + m.err.Error()))
	case len(m.sessions) == 0:
		m.viewport.SetContent(renderEmpty(m.width, m.viewport.Height()))
	default:
		m.viewport.SetContent(renderList(m.sessions, m.cursor))
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

func renderList(sessions []*sessiondata.Session, cursor int) string {
	var b strings.Builder
	for i, sess := range sessions {
		b.WriteString(renderSession(sess, i == cursor))
		b.WriteString("\n\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderSession(sess *sessiondata.Session, selected bool) string {
	title := sessionTitle(sess)

	cursor := noCursorPrefix

	id := itemStyle.Render(shortSessionID(sess.ID))

	titleText := itemStyle.Render(title)
	if selected {
		cursor = selectedStyle.Render(cursorPrefix)
		id = selectedStyle.Render(shortSessionID(sess.ID))
		titleText = selectedStyle.Render(title)
	}

	titleLine := cursor + id + titleSep + titleText
	metaLine := metaStyle.Render(metaPad + sessionMeta(sess))

	return titleLine + "\n" + metaLine
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
		m.viewport.SetYOffset(itemEnd - height + 1)
	}
}

func (m Model) loadSessions() tea.Cmd {
	return func() tea.Msg {
		sessions, err := m.server.ListSessions(m.ctx)

		return sessionsLoadedMsg{sessions: sessions, err: err}
	}
}

func (m Model) deleteSession(id string) tea.Cmd {
	return func() tea.Msg {
		deleteErr := m.server.DeleteSession(m.ctx, id)

		sessions, err := m.server.ListSessions(m.ctx)
		if err == nil && deleteErr != nil {
			err = deleteErr
		}

		return sessionsLoadedMsg{sessions: sessions, err: err}
	}
}

func sessionMeta(sess *sessiondata.Session) string {
	parts := []string{sess.UpdatedAt.Format(timestampFormat)}
	if cwd := utils.CompactHome(sess.Cwd); cwd != "" {
		parts = append(parts, cwd)
	}

	if tokens := sessionTokens(sess.Usage); tokens != "" {
		parts = append(parts, tokens)
	}

	return strings.Join(parts, metaSep)
}

func sessionTokens(u kit.Usage) string {
	if u.InputTokens == 0 && u.OutputTokens == 0 {
		return ""
	}

	return fmt.Sprintf("%s in / %s out", utils.FormatTokens(u.InputTokens), utils.FormatTokens(u.OutputTokens))
}

func sessionTitle(sess *sessiondata.Session) string {
	if sess.Title != "" {
		return sess.Title
	}

	return untitledText
}

func shortSessionID(id string) string {
	if len(id) >= shortIDLen {
		return id[:shortIDLen]
	}

	return id
}

func clampCursor(cursor, count int) int {
	switch {
	case count <= 0:
		return 0
	case cursor < 0:
		return 0
	case cursor >= count:
		return count - 1
	}

	return cursor
}
