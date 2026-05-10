package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/server"
	sessionScreen "github.com/vitaliiPsl/crappy-ai/internal/tui/screen/session"
)

const paddingX = 2

var contentStyle = lipgloss.NewStyle().PaddingLeft(paddingX).PaddingRight(paddingX)

type Model struct {
	ctx    context.Context
	server *server.Server

	sessionID string
	session   *sessionScreen.Model

	width  int
	height int
}

func New(ctx context.Context, srv *server.Server) Model {
	sess := sessionScreen.New(ctx, srv, "")

	return Model{
		ctx:     ctx,
		server:  srv,
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
		if msg.String() == "ctrl+c" {
			if m.session != nil {
				m.session.Cleanup()
			}

			return m, tea.Quit
		}
	case sessionScreen.CreatedMsg:
		m.sessionID = msg.SessionID

		return m, nil
	}

	if m.session != nil {
		var cmd tea.Cmd

		*m.session, cmd = m.session.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m Model) View() tea.View {
	content := ""
	if m.session != nil {
		content = m.session.View()
	}

	view := tea.NewView(contentStyle.Render(content))
	view.AltScreen = true

	return view
}

func (m *Model) resize() {
	innerWidth := max(m.width-2*paddingX, 0)

	if m.session != nil {
		m.session.SetSize(innerWidth, m.height)
	}
}
