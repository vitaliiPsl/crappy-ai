package mcp

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	coremcp "github.com/vitaliiPsl/crappy-ai/internal/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
)

const (
	headerText    = "MCP Clients"
	hintsText     = "j/Down Move • r Refresh • Esc Back"
	emptyTitle    = "No MCP clients configured"
	emptySubtitle = "Add mcp_clients to settings.yaml."

	cursorPrefix   = "> "
	noCursorPrefix = "  "
	titleSep       = "  "
	metaPad        = "  "
	metaSep        = " · "
	headerLines    = 2
	bottomHeight   = 1
	itemHeight     = 4
)

type Model struct {
	server *server.Server

	statuses []coremcp.ClientStatus
	cursor   int

	viewport viewport.Model
	width    int
	height   int
}

func New(srv *server.Server) Model {
	vp := viewport.New()
	vp.SoftWrap = false

	return Model{
		server:   srv,
		viewport: vp,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadStatuses()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusesLoadedMsg:
		m.statuses = msg.statuses
		m.cursor = clampCursor(m.cursor, len(m.statuses))
		m.refreshContent()
		m.scrollToCursor()

		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	header := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(headerStyle.Render(headerText))
	bottom := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(hintsStyle.Render(hintsText))

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
	case "r":
		return m, m.loadStatuses()
	case "esc":
		return m, func() tea.Msg { return ClosedMsg{} }
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(key)

	return m, cmd
}

func (m Model) moveCursor(delta int) (Model, tea.Cmd) {
	next := m.cursor + delta
	if next < 0 || next >= len(m.statuses) {
		return m, nil
	}

	m.cursor = next
	m.refreshContent()
	m.scrollToCursor()

	return m, nil
}

func (m *Model) resizeViewport() {
	m.viewport.SetHeight(max(m.height-headerLines-bottomHeight, 1))
}

func (m *Model) refreshContent() {
	if len(m.statuses) == 0 {
		m.viewport.SetContent(renderEmpty(m.width, m.viewport.Height()))

		return
	}

	m.viewport.SetContent(renderList(m.statuses, m.cursor))
}

func (m Model) loadStatuses() tea.Cmd {
	return func() tea.Msg {
		return statusesLoadedMsg{statuses: m.server.GetMCPClientStatuses()}
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

func renderList(statuses []coremcp.ClientStatus, cursor int) string {
	var b strings.Builder
	for i, status := range statuses {
		b.WriteString(renderStatus(status, i == cursor))
		b.WriteString("\n\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderStatus(status coremcp.ClientStatus, selected bool) string {
	cursor := noCursorPrefix

	name := status.Config.Name
	if name == "" {
		name = "(unnamed)"
	}

	title := itemStyle.Render(name)
	if selected {
		cursor = selectedStyle.Render(cursorPrefix)
		title = selectedStyle.Render(name)
	}

	lines := []string{
		cursor + statusBadge(status.State) + titleSep + title,
		metaStyle.Render(metaPad + strings.Join(statusMeta(status), metaSep)),
	}

	if status.Error != "" {
		lines = append(lines, errorStyle.Render(metaPad+status.Error))
	}

	return strings.Join(lines, "\n")
}

func statusMeta(status coremcp.ClientStatus) []string {
	cfg := status.Config

	transport := cfg.Transport
	if transport == "" {
		transport = coremcp.TransportStdio
	}

	parts := []string{string(transport)}
	switch transport {
	case coremcp.TransportHTTP:
		if cfg.URL != "" {
			parts = append(parts, cfg.URL)
		}
	default:
		if cfg.Command != "" {
			parts = append(parts, strings.Join(append([]string{cfg.Command}, cfg.Args...), " "))
		}
	}

	if authCount := len(cfg.Auth.Headers) + len(cfg.Auth.HeaderEnv); authCount > 0 {
		parts = append(parts, fmt.Sprintf("%d auth header(s)", authCount))
	}

	return parts
}

func statusBadge(state coremcp.ClientState) string {
	switch state {
	case coremcp.ClientConnected:
		return successStyle.Render("connected")
	case coremcp.ClientConnecting:
		return warningStyle.Render("connecting")
	case coremcp.ClientFailed:
		return errorStyle.Render("failed")
	default:
		return disconnectedStyle.Render("disconnected")
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
		m.viewport.SetYOffset(itemEnd - height + 1)
	}
}

func clampCursor(cursor, count int) int {
	switch {
	case count <= 0:
		return 0
	case cursor < 0:
		return 0
	case cursor >= count:
		return count - 1
	default:
		return cursor
	}
}
