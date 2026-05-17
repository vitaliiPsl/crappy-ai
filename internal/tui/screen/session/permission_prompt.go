package session

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

const (
	promptPaddingX    = 1
	promptHintsText   = "y Once • g Global • n Deny"
	promptLabelMaxLen = 60
)

type permissionPromptMsg struct {
	ToolCallID string
	Response   permission.Response
}

type permissionPrompt struct {
	call  kit.ToolCall
	width int
}

func newPermissionPrompt(call kit.ToolCall) permissionPrompt {
	return permissionPrompt{call: call}
}

func (p permissionPrompt) Update(msg tea.Msg) (permissionPrompt, tea.Cmd, tea.Msg) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil, nil
	}

	switch key.String() {
	case "y", "enter":
		return p, nil, p.emit(permission.Allow, permission.ScopeOnce)
	case "g":
		return p, nil, p.emit(permission.Allow, permission.ScopeGlobal)
	case "n", "esc":
		return p, nil, p.emit(permission.Deny, permission.ScopeOnce)
	}

	return p, nil, nil
}

func (p permissionPrompt) View() string {
	thm := theme.Default

	prompt := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().
			Foreground(thm.Primary).
			Background(thm.SurfaceAlt).
			Render(inputPrompt),
		lipgloss.NewStyle().
			Foreground(thm.Warning).
			Background(thm.SurfaceAlt).
			Render(fmt.Sprintf("Allow %s?", toolLabel(p.call))),
	)

	box := lipgloss.NewStyle().
		Width(p.width).
		Background(thm.SurfaceAlt).
		Padding(0, promptPaddingX)

	return strings.TrimRight(box.Render("\n"+prompt+"\n"), "\n")
}

func (p permissionPrompt) HintsText() string {
	return promptHintsText
}

func (p permissionPrompt) Height() int {
	return lipgloss.Height(p.View())
}

func (p *permissionPrompt) SetWidth(width int) {
	p.width = width
}

func (p permissionPrompt) emit(decision permission.Decision, scope permission.Scope) tea.Msg {
	return permissionPromptMsg{
		ToolCallID: p.call.ID,
		Response:   permission.Response{Decision: decision, Scope: scope},
	}
}

func toolLabel(call kit.ToolCall) string {
	switch {
	case argString(call, "command") != "":
		return fmt.Sprintf("%s: $ %s", call.Name, truncateLabel(argString(call, "command")))
	case argString(call, "path") != "":
		return fmt.Sprintf("%s: %s", call.Name, argString(call, "path"))
	case argString(call, "url") != "":
		return fmt.Sprintf("%s: %s", call.Name, argString(call, "url"))
	default:
		return call.Name
	}
}

func argString(call kit.ToolCall, key string) string {
	v, _ := call.Arguments[key].(string)

	return v
}

func truncateLabel(s string) string {
	if len(s) <= promptLabelMaxLen {
		return s
	}

	return s[:promptLabelMaxLen] + titleEllipsis
}
