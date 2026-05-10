package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/server"
)

const paddingX = 2

var contentStyle = lipgloss.NewStyle().PaddingLeft(paddingX).PaddingRight(paddingX)

type Model struct {
	ctx    context.Context
	server *server.Server

	width  int
	height int
}

func New(ctx context.Context, srv *server.Server) Model {
	return Model{ctx: ctx, server: srv}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() tea.View {
	view := tea.NewView(contentStyle.Render("crappy-ai\n\nctrl+c quit"))
	view.AltScreen = true

	return view
}
