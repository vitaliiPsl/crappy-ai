package session

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
)

const (
	titleMaxLen   = 30
	titleEllipsis = "..."
)

func (m Model) handleKey(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		if m.state.Prompt != nil {
			return m.handlePromptKey(key)
		}

		if m.state.Phase == PhaseRunning {
			m.server.CancelRun(m.state.ID)

			return m, nil
		}

		if m.input.PickerActive() {
			m.input.ClearPicker()

			return m, nil
		}

		m.input.Reset()

		return m, nil

	case "ctrl+o":
		m.showThinking = !m.showThinking

		return m, nil

	case "ctrl+t":
		m.showToolResult = !m.showToolResult

		return m, nil

	case "up", "down":
		if m.input.PickerActive() {
			var cmd tea.Cmd

			m.input, cmd = m.input.Update(key)

			return m, cmd
		}

		var cmd tea.Cmd

		m.viewport, cmd = m.viewport.Update(key)

		return m, cmd

	case "pgup", "pgdown", "ctrl+u", "ctrl+d":
		var cmd tea.Cmd

		m.viewport, cmd = m.viewport.Update(key)

		return m, cmd
	}

	switch focusForState(m.state) {
	case FocusPermissionPrompt:
		return m.handlePromptKey(key)

	case FocusInput:
		var cmd tea.Cmd

		m.input, cmd = m.input.Update(key)

		return m, cmd
	}

	return m, nil
}

func (m Model) handleCommand(msg commandMsg) (Model, tea.Cmd) {
	cmdDef, ok := m.registry.Get(msg.Name)
	if !ok {
		m.state = m.state.AppendSystem(fmt.Sprintf("Unknown command: /%s", msg.Name))

		return m, nil
	}

	return m, cmdDef.Execute(m.ctx, command.Request{SessionID: m.state.ID, Args: msg.Args})
}

func (m Model) handleCompact() (Model, tea.Cmd) {
	if m.state.Phase != PhaseIdle || m.state.ID == "" {
		return m, nil
	}

	m.state = m.state.StartTurn()

	return m, tea.Batch(compactCmd(m.ctx, m.server, m.state.ID), m.spinner.Tick)
}

func (m Model) handlePromptKey(key tea.KeyMsg) (Model, tea.Cmd) {
	req := m.state.Prompt
	if req == nil {
		return m, nil
	}

	optionID := pickPromptOption(key, *req)
	if optionID == "" {
		return m, nil
	}

	toolCallID := req.Call.ID
	m.state = m.state.AnswerPrompt()

	return m, respondPromptCmd(m.server, m.state.ID, toolCallID, model.AskResponse{OptionID: optionID})
}

func (m Model) handleSubmit(text string) (Model, tea.Cmd) {
	if m.state.Phase != PhaseIdle {
		return m, nil
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return m, nil
	}

	var cmds []tea.Cmd

	if m.state.ID == "" {
		sess, ch, err := m.openNewSession(deriveTitle(text))
		if err != nil {
			m.state = m.state.SetError(err)

			return m, nil
		}

		m.state = m.state.WithSession(sess)
		m.events = ch

		cmds = append(cmds, announceCreated(sess.ID), waitForEventCmd(ch))
	}

	m.state = m.state.StartTurn()
	cmds = append(cmds, sendCmd(m.ctx, m.server, m.state.ID, text), m.spinner.Tick)

	return m, tea.Batch(cmds...)
}

func (m Model) applyHistory(msg historyLoadedMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.state = m.state.SetError(msg.err)
		m.refresh()

		return m, nil
	}

	m.state = m.state.Reset()

	for _, ev := range msg.events {
		m.state = Reduce(m.state, ev)
	}

	m.refresh()
	m.viewport.GotoBottom()

	return m, nil
}

func (m Model) applyEvent(ev sessiondata.Event) (Model, tea.Cmd) {
	wasRunning := m.state.Phase == PhaseRunning || m.state.Phase == PhaseCompacting
	follow := m.isNearBottom()

	m.state = Reduce(m.state, ev)
	m.refresh()

	if follow {
		m.viewport.GotoBottom()
	}

	var cmds []tea.Cmd

	cmds = append(cmds, waitForEventCmd(m.events))

	nowRunning := m.state.Phase == PhaseRunning || m.state.Phase == PhaseCompacting
	if !wasRunning && nowRunning {
		cmds = append(cmds, m.spinner.Tick)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) tickSpinner(msg spinner.TickMsg) (Model, tea.Cmd) {
	if m.state.Phase != PhaseRunning && m.state.Phase != PhaseCompacting {
		return m, nil
	}

	var cmd tea.Cmd

	m.spinner, cmd = m.spinner.Update(msg)

	return m, cmd
}

func (m Model) openNewSession(title string) (*sessiondata.Session, <-chan sessiondata.Event, error) {
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

func announceCreated(sessionID string) tea.Cmd {
	return func() tea.Msg { return CreatedMsg{SessionID: sessionID} }
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
