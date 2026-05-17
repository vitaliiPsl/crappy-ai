package session

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
)

const (
	titleMaxLen   = 30
	titleEllipsis = "..."
)

type Model struct {
	ctx    context.Context
	server *server.Server
	sess   *sessiondata.Session
	reg    *command.Registry

	conversation conversation
	footer       footer
	eventChan    <-chan sessiondata.Event

	err       error
	runActive bool

	width  int
	height int
}

func New(ctx context.Context, srv *server.Server, sessionID string) Model {
	cfg := srv.GetConfig()
	reg := newCommandRegistry()

	sess, eventChan, initErr := openInitialSession(ctx, srv, sessionID)

	cwd := cfg.Cwd
	if sess != nil && sess.Cwd != "" {
		cwd = sess.Cwd
	}

	return Model{
		ctx:          ctx,
		server:       srv,
		reg:          reg,
		conversation: newConversation(cfg.Provider, cfg.Model),
		footer:       newFooter(reg, cfg.Model, cwd),
		sess:         sess,
		eventChan:    eventChan,
		err:          initErr,
	}
}

func newCommandRegistry() *command.Registry {
	registry := command.NewRegistry()
	registry.Register(command.NewNewCommand())
	registry.Register(command.NewSessionsCommand())
	registry.Register(command.NewSettingsCommand())
	registry.Register(command.NewCompactCommand())
	registry.Register(command.NewHelpCommand(registry))

	return registry
}

func openInitialSession(
	ctx context.Context,
	srv *server.Server,
	sessionID string,
) (*sessiondata.Session, <-chan sessiondata.Event, error) {
	if sessionID == "" {
		return nil, nil, nil
	}

	sess, err := srv.GetSession(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}

	eventChan, err := srv.Subscribe(ctx, sessionID)
	if err != nil {
		return sess, nil, err
	}

	return sess, eventChan, nil
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.footer.Init()}

	if m.sessionID() != "" {
		cmds = append(cmds, m.loadHistory(), m.waitForEvent())
	}

	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case historyLoadedMsg:
		if msg.err != nil {
			m.err = msg.err

			return m, nil
		}

		return m.updateChildren(msg)

	case sessionEventMsg:
		switch msg.event.Type {
		case sessiondata.EventTurnComplete, sessiondata.EventTurnCancelled, sessiondata.EventError:
			m.runActive = false
		}

		var cmd tea.Cmd

		m, cmd = m.updateChildren(msg)

		return m, tea.Batch(cmd, m.waitForEvent())

	case permissionPromptMsg:
		return m, m.respondPrompt(msg.ToolCallID, msg.Response)

	case runStartedMsg:
		m.runActive = true

		return m.updateChildren(msg)

	case errorMsg:
		m.runActive = false
		m.err = msg.err

		return m.updateChildren(runStoppedMsg{})

	case submitMsg:
		if m.runActive {
			return m, nil
		}

		return m.handleSubmit(msg.Text)

	case commandMsg:
		if m.runActive {
			return m, nil
		}

		return m.runCommand(msg)

	case command.SystemMsg:
		var cmd tea.Cmd

		m, cmd = m.updateChildren(systemMessageMsg{Text: msg.Text})

		return m, cmd

	case command.CompactSessionMsg:
		if m.runActive || m.sessionID() == "" {
			return m, nil
		}

		return m, m.runCompact()

	case tea.KeyMsg:
		if msg.String() == "esc" && m.runActive && !m.footer.HasPrompt() {
			m.server.CancelRun(m.sessionID())

			return m, nil
		}
	}

	return m.updateChildren(msg)
}

func (m Model) respondPrompt(toolCallID string, resp permission.Response) tea.Cmd {
	return func() tea.Msg {
		if err := m.server.RespondPrompt(m.sessionID(), toolCallID, resp); err != nil {
			return errorMsg{err: err}
		}

		return nil
	}
}

func (m Model) runCommand(msg commandMsg) (Model, tea.Cmd) {
	m.err = nil

	cmd, ok := m.reg.Get(msg.Name)
	if !ok {
		m, updateCmd := m.updateChildren(systemMessageMsg{Text: fmt.Sprintf("Unknown command: /%s", msg.Name)})

		return m, updateCmd
	}

	return m, cmd.Execute(m.ctx, command.Request{SessionID: m.sessionID(), Args: msg.Args})
}

func (m Model) View() string {
	errView := ""
	if m.err != nil {
		errView = errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
	}

	return m.conversation.View() + "\n" + errView + m.footer.View()
}

func (m Model) updateChildren(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	var (
		cmd      tea.Cmd
		consumed bool
	)

	m.footer, cmd, consumed = m.footer.Update(msg)
	cmds = append(cmds, cmd)

	if !consumed {
		m.conversation, cmd = m.conversation.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.SetSize(m.width, m.height)

	return m, tea.Batch(cmds...)
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.footer.setSize(width)

	footerHeight := m.footer.Height()
	convHeight := max(height-footerHeight, 1)
	m.conversation.setSize(width, convHeight)
}

func (m *Model) Cleanup() {
	if m.eventChan != nil {
		m.server.Unsubscribe(m.sessionID(), m.eventChan)
		m.eventChan = nil
	}
}

func (m Model) sessionID() string {
	if m.sess == nil {
		return ""
	}

	return m.sess.ID
}

func (m Model) loadHistory() tea.Cmd {
	return func() tea.Msg {
		events, err := m.server.LoadEvents(m.ctx, m.sessionID())

		return historyLoadedMsg{events: events, err: err}
	}
}

func (m Model) waitForEvent() tea.Cmd {
	ch := m.eventChan
	if ch == nil {
		return nil
	}

	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}

		return sessionEventMsg{event: ev}
	}
}

func (m Model) startRun(text string) tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return runStartedMsg{} },
		func() tea.Msg {
			if err := m.server.Send(m.ctx, m.sessionID(), text); err != nil {
				return errorMsg{err: err}
			}

			return nil
		},
	)
}

func (m Model) runCompact() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return runStartedMsg{} },
		func() tea.Msg {
			if err := m.server.Compact(m.ctx, m.sessionID()); err != nil {
				return errorMsg{err: err}
			}

			return nil
		},
	)
}

func (m Model) handleSubmit(text string) (Model, tea.Cmd) {
	m.err = nil

	var cmds []tea.Cmd
	if m.sessionID() == "" {
		sess, ch, err := m.createSession(deriveTitle(text))
		if err != nil {
			m.err = err

			return m, nil
		}

		m.sess = sess
		m.eventChan = ch

		cmds = append(cmds, func() tea.Msg { return CreatedMsg{SessionID: sess.ID} })
		cmds = append(cmds, m.waitForEvent())
	}

	cmds = append(cmds, m.startRun(text))

	return m, tea.Batch(cmds...)
}

func (m Model) createSession(title string) (*sessiondata.Session, <-chan sessiondata.Event, error) {
	sess, err := m.server.CreateSession(m.ctx, title)
	if err != nil {
		return nil, nil, err
	}

	ch, err := m.server.Subscribe(m.ctx, sess.ID)
	if err != nil {
		return nil, nil, err
	}

	return sess, ch, nil
}

func deriveTitle(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= titleMaxLen {
		return text
	}

	trimmed := text[:titleMaxLen]
	if idx := strings.LastIndex(trimmed, " "); idx > 0 {
		trimmed = trimmed[:idx]
	}

	return trimmed + titleEllipsis
}
