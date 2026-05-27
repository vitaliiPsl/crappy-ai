package session

import (
	"context"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/server"
	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
)

const followBottomGrip = 2

type Model struct {
	ctx    context.Context
	server *server.Server

	state    State
	events   <-chan sessiondata.Event
	registry *command.Registry

	viewport viewport.Model
	spinner  spinner.Model
	input    inputBar

	showThinking   bool
	showToolResult bool

	width  int
	height int
}

func New(ctx context.Context, srv *server.Server, sessionID string) Model {
	cfg := srv.GetConfig()
	registry := command.NewRegistry(srv)

	vp := viewport.New()
	vp.SoftWrap = true

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = spinnerStyle

	m := Model{
		ctx:      ctx,
		server:   srv,
		state:    NewState(cfg),
		registry: registry,
		viewport: vp,
		spinner:  sp,
		input:    newInputBar(registry),
	}

	if sessionID == "" {
		return m
	}

	sess, err := srv.GetSession(ctx, sessionID)
	if err != nil {
		m.state = m.state.SetError(err)

		return m
	}

	ch, err := srv.Subscribe(ctx, sessionID)
	if err != nil {
		m.state = m.state.SetError(err)

		return m
	}

	m.state = m.state.WithSession(sess)
	m.events = ch

	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.input.Init()}

	if m.state.ID != "" {
		cmds = append(cmds, loadHistoryCmd(m.ctx, m.server, m.state.ID), waitForEventCmd(m.events))
	}

	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)

		return m, nil

	case spinner.TickMsg:
		return m.tickSpinner(msg)

	case sessionEventMsg:
		return m.applyEvent(msg.event)

	case historyLoadedMsg:
		return m.applyHistory(msg)

	case submitMsg:
		m, cmd = m.handleSubmit(msg.Text)

	case commandMsg:
		m, cmd = m.handleCommand(msg)

	case command.SystemMsg:
		m.state = m.state.AppendSystem(msg.Text)

	case command.SubmitTextMsg:
		m, cmd = m.handleSubmit(msg.Text)

	case command.SubmitSkillMsg:
		m, cmd = m.handleSkillSubmit(msg)

	case command.CompactSessionMsg:
		m, cmd = m.handleCompact()

	case modeUpdatedMsg:
		m = m.handleModeUpdated(msg)

	case effectErrorMsg:
		m.state = m.state.SetError(msg.err)

	case tea.KeyMsg:
		m, cmd = m.handleKey(msg)

	default:
		m.viewport, cmd = m.viewport.Update(msg)

		return m, cmd
	}

	m.refresh()

	return m, cmd
}

func (m Model) View() string {
	return m.viewport.View() + "\n" + m.renderFooterView()
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.SetWidth(width)
	m.input.SetWidth(width)
	m.refresh()
}

func (m *Model) Cleanup() {
	if m.events == nil {
		return
	}

	m.server.Unsubscribe(m.state.ID, m.events)
	m.events = nil
}

func (m *Model) refresh() {
	footerHeight := lipgloss.Height(m.renderFooterView())
	bodyHeight := max(m.height-footerHeight, 1)
	m.viewport.SetHeight(bodyHeight)

	if isConversationEmpty(&m.state) {
		m.viewport.SetContent(renderEmpty(&m.state, m.width, bodyHeight))

		return
	}

	opts := ConvOpts{
		Width:          m.width,
		ShowThinking:   m.showThinking,
		ShowToolResult: m.showToolResult,
	}

	m.viewport.SetContent(renderConversation(&m.state, opts))
}

func (m Model) renderFooterView() string {
	return renderFooter(&m.state, FooterOpts{
		Width:   m.width,
		Spinner: m.spinner.View(),
		Input:   m.input.View(),
	})
}

func (m Model) isNearBottom() bool {
	total := m.viewport.TotalLineCount()
	if total <= m.viewport.Height() {
		return true
	}

	return m.viewport.YOffset()+m.viewport.Height() >= total-followBottomGrip
}
