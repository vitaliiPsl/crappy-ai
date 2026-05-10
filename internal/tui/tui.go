package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/server"
	sessionScreen "github.com/vitaliiPsl/crappy-ai/internal/tui/screen/session"
	sessionsScreen "github.com/vitaliiPsl/crappy-ai/internal/tui/screen/sessions"
)

const paddingX = 2

var contentStyle = lipgloss.NewStyle().PaddingLeft(paddingX).PaddingRight(paddingX)

type screen int

const (
	screenSession screen = iota
	screenSessions
)

type Model struct {
	ctx    context.Context
	server *server.Server

	active screen
	prev   screen

	sessionID string
	session   *sessionScreen.Model
	sessions  *sessionsScreen.Model

	width  int
	height int
}

func New(ctx context.Context, srv *server.Server) Model {
	sess := sessionScreen.New(ctx, srv, "")

	return Model{
		ctx:     ctx,
		server:  srv,
		active:  screenSession,
		session: &sess,
	}
}

func (m Model) Init() tea.Cmd {
	if m.session != nil {
		return m.session.Init()
	}

	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()

		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.session != nil {
				m.session.Cleanup()
			}

			return m, tea.Quit

		case "ctrl+p":
			if m.active == screenSession && m.session != nil {
				m.session.Cleanup()
			}

			return m, m.openSessions()

		case "ctrl+n":
			if m.session != nil {
				m.session.Cleanup()
			}

			return m, m.openSession("")
		}
	case sessionScreen.CreatedMsg:
		m.sessionID = msg.SessionID

		return m, nil
	case sessionsScreen.OpenSessionMsg:
		return m, m.openSession(msg.SessionID)
	case sessionsScreen.OpenDraftSessionMsg:
		return m, m.openSession("")
	case sessionsScreen.ClosedMsg:
		return m, m.openPrev()
	}

	switch m.active {
	case screenSession:
		if m.session == nil {
			return m, nil
		}

		var cmd tea.Cmd

		*m.session, cmd = m.session.Update(msg)

		return m, cmd

	case screenSessions:
		if m.sessions == nil {
			return m, nil
		}

		var cmd tea.Cmd

		*m.sessions, cmd = m.sessions.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m Model) View() tea.View {
	content := ""

	switch m.active {
	case screenSession:
		if m.session != nil {
			content = m.session.View()
		}
	case screenSessions:
		if m.sessions != nil {
			content = m.sessions.View()
		}
	}

	view := tea.NewView(contentStyle.Render(content))
	view.AltScreen = true

	return view
}

func (m *Model) resize() {
	innerWidth := max(m.width-2*paddingX, 0)

	switch m.active {
	case screenSession:
		if m.session == nil {
			return
		}

		m.session.SetSize(innerWidth, m.height)
	case screenSessions:
		if m.sessions == nil {
			return
		}

		m.sessions.SetSize(innerWidth, m.height)
	}
}

func (m *Model) openSession(sessionID string) tea.Cmd {
	sess := sessionScreen.New(m.ctx, m.server, sessionID)
	m.session = &sess
	m.sessionID = sessionID
	m.active = screenSession
	m.resize()

	return m.session.Init()
}

func (m *Model) openSessions() tea.Cmd {
	m.prev = m.active
	ss := sessionsScreen.New(m.ctx, m.server)
	m.sessions = &ss
	m.active = screenSessions
	m.resize()

	return m.sessions.Init()
}

func (m *Model) openPrev() tea.Cmd {
	switch m.prev {
	case screenSession:
		return m.openSession(m.sessionID)
	default:
		return m.openSession("")
	}
}
