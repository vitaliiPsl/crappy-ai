package sessions

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/server"
	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

const (
	headerText      = "Sessions"
	emptyTitle      = "No sessions yet"
	emptySubtitle   = "Press n to start a new conversation."
	hintsText       = "j/Down Move • Enter Open • n New • d Delete • r Refresh • Esc Back"
	timestampFormat = "Jan 02 15:04"

	cursorPrefix   = "> "
	noCursorPrefix = "  "
	timestampPad   = "  "
	untitledText   = "Untitled session"
	errorPrefix    = "Error: "

	shortIDLen  = 8
	itemHeight  = 3
	headerLines = 2
	hintsHeight = 1
)

var (
	thm = theme.Default

	headerStyle    = lipgloss.NewStyle().Foreground(thm.Primary).Bold(true)
	selectedStyle  = lipgloss.NewStyle().Foreground(thm.Primary).Bold(true)
	itemStyle      = lipgloss.NewStyle().Foreground(thm.Text)
	timestampStyle = lipgloss.NewStyle().Foreground(thm.SubtleText)
	emptyStyle     = lipgloss.NewStyle().Foreground(thm.SubtleText)
	errorStyle     = lipgloss.NewStyle().Foreground(thm.Error)
	hintsStyle     = lipgloss.NewStyle().Foreground(thm.SubtleText)
)

type Model struct {
	ctx    context.Context
	server *server.Server

	sessions []*sessiondata.Session
	cursor   int
	err      error

	viewport viewport.Model

	width  int
	height int
}

func New(ctx context.Context, srv *server.Server) Model {
	vp := viewport.New()
	vp.SoftWrap = false

	return Model{
		ctx:      ctx,
		server:   srv,
		viewport: vp,
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
		m.cursor = clampCursor(msg.cursor, len(m.sessions))
		m.refreshContent()
		m.scrollToCursor()

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.refreshContent()
				m.scrollToCursor()
			}

			return m, nil

		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
				m.refreshContent()
				m.scrollToCursor()
			}

			return m, nil

		case "enter":
			if len(m.sessions) == 0 {
				return m, nil
			}

			sessionID := m.sessions[m.cursor].ID

			return m, func() tea.Msg { return OpenSessionMsg{SessionID: sessionID} }

		case "n":
			return m, func() tea.Msg { return OpenDraftSessionMsg{} }

		case "d":
			if len(m.sessions) == 0 {
				return m, nil
			}

			return m, m.deleteSession(m.sessions[m.cursor].ID)

		case "r":
			return m, m.loadSessions()

		case "esc":
			return m, func() tea.Msg { return ClosedMsg{} }
		}
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	header := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(headerStyle.Render(headerText))
	hints := hintsStyle.Width(m.width).Align(lipgloss.Center).Render(hintsText)

	return header + "\n\n" + m.viewport.View() + "\n" + hints
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(max(height-headerLines-hintsHeight, 1))
	m.refreshContent()
}

func (m *Model) refreshContent() {
	if m.err != nil {
		m.viewport.SetContent(errorStyle.Render(errorPrefix + m.err.Error()))

		return
	}

	if len(m.sessions) == 0 {
		m.viewport.SetContent(m.renderEmpty())

		return
	}

	var b strings.Builder
	for i, sess := range m.sessions {
		title := sessionTitle(sess)
		timestamp := sess.UpdatedAt.Format(timestampFormat)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(cursorPrefix + title))
		} else {
			b.WriteString(noCursorPrefix + itemStyle.Render(title))
		}

		b.WriteByte('\n')
		b.WriteString(timestampStyle.Render(timestampPad + timestamp))
		b.WriteString("\n\n")
	}

	m.viewport.SetContent(strings.TrimRight(b.String(), "\n"))
}

func (m Model) renderEmpty() string {
	content := emptyStyle.Render(emptyTitle) + "\n" + emptyStyle.Render(emptySubtitle)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(max(m.viewport.Height(), 1)).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(content)
}

func (m *Model) scrollToCursor() {
	itemStart := m.cursor * itemHeight
	itemEnd := itemStart + itemHeight - 1
	vpHeight := m.viewport.Height()
	offset := m.viewport.YOffset()

	if itemStart < offset {
		m.viewport.SetYOffset(itemStart)
	} else if itemEnd >= offset+vpHeight {
		m.viewport.SetYOffset(itemEnd - vpHeight + 1)
	}
}

func (m Model) loadSessions() tea.Cmd {
	return func() tea.Msg {
		sessions, err := m.server.ListSessions(m.ctx)

		return sessionsLoadedMsg{sessions: sessions, err: err, cursor: m.cursor}
	}
}

func (m Model) deleteSession(sessionID string) tea.Cmd {
	nextCursor := m.cursor

	return func() tea.Msg {
		deleteErr := m.server.DeleteSession(m.ctx, sessionID)

		sessions, err := m.server.ListSessions(m.ctx)
		if err == nil && deleteErr != nil {
			err = deleteErr
		}

		return sessionsLoadedMsg{sessions: sessions, err: err, cursor: nextCursor}
	}
}

func sessionTitle(sess *sessiondata.Session) string {
	if sess.Title != "" {
		return sess.Title
	}

	if len(sess.ID) >= shortIDLen {
		return fmt.Sprintf("%s %s", untitledText, sess.ID[:shortIDLen])
	}

	return untitledText
}

func clampCursor(cursor, count int) int {
	if count <= 0 {
		return 0
	}

	if cursor < 0 {
		return 0
	}

	if cursor >= count {
		return count - 1
	}

	return cursor
}
