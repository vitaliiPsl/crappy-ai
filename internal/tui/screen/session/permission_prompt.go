package session

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

const (
	promptPaddingX    = 1
	promptLabelMaxLen = 60
)

type permissionPromptMsg struct {
	ToolCallID string
	Response   model.AskResponse
}

type permissionPrompt struct {
	request model.AskRequest
	width   int
}

func newPermissionPrompt(request model.AskRequest) permissionPrompt {
	return permissionPrompt{request: request}
}

func (p permissionPrompt) Update(msg tea.Msg) (permissionPrompt, tea.Cmd, tea.Msg) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil, nil
	}

	switch key.String() {
	case "y", "enter":
		return p, nil, p.emit(model.OptionAllowOnce)
	case "e":
		return p, nil, p.emit(model.OptionAllowExact)
	case "g":
		return p, nil, p.emit(model.OptionAllowPattern)
	case "n", "esc":
		return p, nil, p.emit(model.OptionDenyOnce)
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
			Render(fmt.Sprintf("Allow %s?", toolLabel(p.request.Call))),
	)

	box := lipgloss.NewStyle().
		Width(p.width).
		Background(thm.SurfaceAlt).
		Padding(0, promptPaddingX)

	return strings.TrimRight(box.Render("\n"+prompt+"\n"), "\n")
}

func (p permissionPrompt) HintsText() string {
	hints := []string{"y Once"}
	if _, ok := p.request.Option(model.OptionAllowExact); ok {
		hints = append(hints, "e Exact")
	}

	if option, ok := p.request.Option(model.OptionAllowPattern); ok {
		hints = append(hints, p.patternHint(option, hints))
	}

	hints = append(hints, "n Deny")

	return strings.Join(hints, " • ")
}

func (p permissionPrompt) patternHint(option model.AskOption, previous []string) string {
	const label = "g Pattern"
	if option.Rule == nil || option.Rule.Pattern == "" {
		return label
	}

	prefix := label + ": "

	available := p.width - lipgloss.Width(strings.Join(append(append([]string{}, previous...), prefix, "n Deny"), " • "))
	if available <= len(titleEllipsis) {
		return label
	}

	return prefix + truncateWidth(option.Rule.Pattern, available)
}

func (p permissionPrompt) Height() int {
	return lipgloss.Height(p.View())
}

func (p *permissionPrompt) SetWidth(width int) {
	p.width = width
}

func (p permissionPrompt) emit(optionID string) tea.Msg {
	if _, ok := p.request.Option(optionID); !ok {
		return nil
	}

	return permissionPromptMsg{
		ToolCallID: p.request.Call.ID,
		Response:   model.AskResponse{OptionID: optionID},
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

func truncateWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}

	if lipgloss.Width(s) <= width {
		return s
	}

	if width <= len(titleEllipsis) {
		return titleEllipsis[:width]
	}

	runes := []rune(s)

	limit := width - len(titleEllipsis)
	if limit > len(runes) {
		limit = len(runes)
	}

	return string(runes[:limit]) + titleEllipsis
}
