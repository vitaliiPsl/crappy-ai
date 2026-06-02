package mcp

import (
	"context"
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
	hintsText     = "j/Down Move • r Refresh • c Reconnect • Esc Back"
	emptyTitle    = "No MCP clients configured"
	emptySubtitle = "Add mcp to settings.yaml."

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

	configs []coremcp.Config
	states  []coremcp.ClientState
	cursor  int

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
	return tea.Batch(m.loadConfigs(), m.loadStates())
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case configsLoadedMsg:
		m.configs = msg.configs
		m.cursor = clampCursor(m.cursor, len(m.configs))
		m.refreshContent()
		m.scrollToCursor()

		return m, nil

	case statesLoadedMsg:
		m.states = msg.states
		m.refreshContent()

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
		return m, m.loadStates()
	case "c":
		return m.reconnectSelected()
	case "esc":
		return m, func() tea.Msg { return ClosedMsg{} }
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(key)

	return m, cmd
}

func (m Model) moveCursor(delta int) (Model, tea.Cmd) {
	next := m.cursor + delta
	if next < 0 || next >= len(m.configs) {
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
	if len(m.configs) == 0 {
		m.viewport.SetContent(renderEmpty(m.width, m.viewport.Height()))

		return
	}

	m.viewport.SetContent(renderList(m.configs, m.states, m.cursor))
}

func (m Model) loadConfigs() tea.Cmd {
	return func() tea.Msg {
		return configsLoadedMsg{configs: m.server.GetMCPClientConfigs()}
	}
}

func (m Model) loadStates() tea.Cmd {
	return func() tea.Msg {
		return statesLoadedMsg{states: m.server.GetMCPClientStates()}
	}
}

func (m Model) reconnectSelected() (Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.configs) {
		return m, nil
	}

	cfg := m.configs[m.cursor]
	if !cfg.IsEnabled() {
		return m, nil
	}

	// Flip the badge to "connecting" right away so the keypress visibly does
	// something; the reconnect resolves it to connected/failed when it lands.
	if m.cursor < len(m.states) {
		m.states[m.cursor] = coremcp.ClientState{Status: coremcp.ClientConnecting}
		m.refreshContent()
	}

	return m, func() tea.Msg {
		_ = m.server.ReconnectMCPClient(context.Background(), cfg.Name)

		return statesLoadedMsg{states: m.server.GetMCPClientStates()}
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

func renderList(configs []coremcp.Config, states []coremcp.ClientState, cursor int) string {
	var b strings.Builder
	for i, cfg := range configs {
		var state coremcp.ClientState
		if i < len(states) {
			state = states[i]
		}

		b.WriteString(renderClient(cfg, state, i == cursor))
		b.WriteString("\n\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderClient(cfg coremcp.Config, state coremcp.ClientState, selected bool) string {
	cursor := noCursorPrefix

	name := cfg.Name
	if name == "" {
		name = "(unnamed)"
	}

	title := itemStyle.Render(name)
	if selected {
		cursor = selectedStyle.Render(cursorPrefix)
		title = selectedStyle.Render(name)
	}

	lines := []string{
		cursor + clientBadge(cfg, state) + titleSep + title,
		metaStyle.Render(metaPad + strings.Join(statusMeta(cfg), metaSep)),
	}

	if state.Error != "" {
		lines = append(lines, errorStyle.Render(metaPad+state.Error))
	}

	return strings.Join(lines, "\n")
}

func statusMeta(cfg coremcp.Config) []string {
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

	if authCount := len(cfg.Headers) + len(cfg.HeaderEnv); authCount > 0 {
		parts = append(parts, fmt.Sprintf("%d auth header(s)", authCount))
	}

	return parts
}

func clientBadge(cfg coremcp.Config, state coremcp.ClientState) string {
	if !cfg.IsEnabled() {
		return disconnectedStyle.Render("disabled")
	}

	return statusBadge(state.Status)
}

func statusBadge(status coremcp.ClientStatus) string {
	switch status {
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
