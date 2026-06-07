package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/server"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
	jobsScreen "github.com/vitaliiPsl/crappy-ai/internal/tui/screen/jobs"
	mcpScreen "github.com/vitaliiPsl/crappy-ai/internal/tui/screen/mcp"
	sessionScreen "github.com/vitaliiPsl/crappy-ai/internal/tui/screen/session"
	sessionsScreen "github.com/vitaliiPsl/crappy-ai/internal/tui/screen/sessions"
	settingsScreen "github.com/vitaliiPsl/crappy-ai/internal/tui/screen/settings"
)

const paddingX = 2

var contentStyle = lipgloss.NewStyle().PaddingLeft(paddingX).PaddingRight(paddingX)

type screen int

const (
	screenSession screen = iota
	screenSessions
	screenSettings
	screenMCP
	screenJobs
)

type Model struct {
	ctx    context.Context
	server *server.Server

	active screen
	prev   screen

	sessionID string
	session   *sessionScreen.Model
	sessions  *sessionsScreen.Model
	settings  *settingsScreen.Model
	mcp       *mcpScreen.Model
	jobs      *jobsScreen.Model

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
	case settingsScreen.ClosedMsg:
		return m, m.openPrev()
	case mcpScreen.ClosedMsg:
		return m, m.openPrev()
	case jobsScreen.ClosedMsg:
		return m, m.openPrev()
	case command.NavNewSessionMsg:
		if m.session != nil {
			m.session.Cleanup()
		}

		return m, m.openSession("")
	case command.NavSessionsMsg:
		if m.session != nil {
			m.session.Cleanup()
		}

		return m, m.openSessions()
	case command.NavSettingsMsg:
		if m.active == screenSession && m.session != nil {
			m.session.Cleanup()
		}

		return m, m.openSettings()
	case command.NavMCPMsg:
		if m.active == screenSession && m.session != nil {
			m.session.Cleanup()
		}

		return m, m.openMCP()
	case command.NavJobsMsg:
		if m.active == screenSession && m.session != nil {
			m.session.Cleanup()
		}

		return m, m.openJobs()
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

	case screenSettings:
		if m.settings == nil {
			return m, nil
		}

		var cmd tea.Cmd

		*m.settings, cmd = m.settings.Update(msg)

		return m, cmd

	case screenMCP:
		if m.mcp == nil {
			return m, nil
		}

		var cmd tea.Cmd

		*m.mcp, cmd = m.mcp.Update(msg)

		return m, cmd

	case screenJobs:
		if m.jobs == nil {
			return m, nil
		}

		var cmd tea.Cmd

		*m.jobs, cmd = m.jobs.Update(msg)

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
	case screenSettings:
		if m.settings != nil {
			content = m.settings.View()
		}
	case screenMCP:
		if m.mcp != nil {
			content = m.mcp.View()
		}
	case screenJobs:
		if m.jobs != nil {
			content = m.jobs.View()
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
	case screenSettings:
		if m.settings == nil {
			return
		}

		m.settings.SetSize(innerWidth, m.height)
	case screenMCP:
		if m.mcp == nil {
			return
		}

		m.mcp.SetSize(innerWidth, m.height)
	case screenJobs:
		if m.jobs == nil {
			return
		}

		m.jobs.SetSize(innerWidth, m.height)
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

func (m *Model) openSettings() tea.Cmd {
	m.prev = m.active
	s := settingsScreen.New(m.server)
	m.settings = &s
	m.active = screenSettings
	m.resize()

	return m.settings.Init()
}

func (m *Model) openMCP() tea.Cmd {
	m.prev = m.active
	screen := mcpScreen.New(m.server)
	m.mcp = &screen
	m.active = screenMCP
	m.resize()

	return m.mcp.Init()
}

func (m *Model) openJobs() tea.Cmd {
	m.prev = m.active
	screen := jobsScreen.New(m.server, m.sessionID)
	m.jobs = &screen
	m.active = screenJobs
	m.resize()

	return m.jobs.Init()
}

func (m *Model) openPrev() tea.Cmd {
	switch m.prev {
	case screenSession:
		return m.openSession(m.sessionID)
	case screenSessions:
		if m.sessions == nil {
			return m.openSessions()
		}

		m.active = screenSessions
		m.resize()

		return nil
	case screenSettings:
		if m.settings == nil {
			return m.openSettings()
		}

		m.active = screenSettings
		m.resize()

		return nil
	case screenMCP:
		if m.mcp == nil {
			return m.openMCP()
		}

		m.active = screenMCP
		m.resize()

		return nil
	case screenJobs:
		if m.jobs == nil {
			return m.openJobs()
		}

		m.active = screenJobs
		m.resize()

		return nil
	default:
		return m.openSession("")
	}
}
