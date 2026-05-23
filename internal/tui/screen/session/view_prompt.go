package session

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
)

const (
	promptPaddingX    = 1
	promptLabelMaxLen = 60
	promptPrompt      = "> "
	promptHintGap     = "g Pattern"
	promptHintGapSep  = ": "
)

func renderPrompt(req *model.AskRequest, width int) string {
	question := fmt.Sprintf("Allow %s?", promptToolLabel(req.Call))

	prefix := promptPrefixStyle.Render(promptPrompt)
	body := promptQuestionStyle.Render(question)
	line := lipgloss.JoinHorizontal(lipgloss.Top, prefix, body)

	return strings.TrimRight(promptBoxStyle.Width(width).Render("\n"+line+"\n"), "\n")
}

func renderPromptHints(req *model.AskRequest, width int) string {
	hints := []string{"y Once"}

	if _, ok := req.Option(model.OptionAllowExact); ok {
		hints = append(hints, "e Exact")
	}

	if option, ok := req.Option(model.OptionAllowPattern); ok {
		hints = append(hints, promptPatternHint(option, hints, width))
	}

	hints = append(hints, "n Deny")

	return strings.Join(hints, " • ")
}

func pickPromptOption(key tea.KeyMsg, req model.AskRequest) string {
	candidate := ""

	switch key.String() {
	case "y", "enter":
		candidate = model.OptionAllowOnce
	case "e":
		candidate = model.OptionAllowExact
	case "g":
		candidate = model.OptionAllowPattern
	case "n", "esc":
		candidate = model.OptionDenyOnce
	}

	if candidate == "" {
		return ""
	}

	if _, ok := req.Option(candidate); !ok {
		return ""
	}

	return candidate
}

func promptPatternHint(option model.AskOption, previous []string, width int) string {
	if option.Rule == nil || option.Rule.Pattern == "" {
		return promptHintGap
	}

	prefix := promptHintGap + promptHintGapSep
	tail := "n Deny"
	available := width - lipgloss.Width(strings.Join(append(append([]string{}, previous...), prefix, tail), " • "))

	if available <= len(ellipsis) {
		return promptHintGap
	}

	return prefix + truncateInlineWidth(option.Rule.Pattern, available)
}

func promptToolLabel(call kit.ToolCall) string {
	switch {
	case promptArg(call, "command") != "":
		return fmt.Sprintf("%s: $ %s", call.Name, truncatePromptLabel(promptArg(call, "command")))
	case promptArg(call, "path") != "":
		return fmt.Sprintf("%s: %s", call.Name, promptArg(call, "path"))
	case promptArg(call, "url") != "":
		return fmt.Sprintf("%s: %s", call.Name, promptArg(call, "url"))
	default:
		return call.Name
	}
}

func promptArg(call kit.ToolCall, key string) string {
	v, _ := call.Arguments[key].(string)

	return v
}

func truncatePromptLabel(s string) string {
	if len(s) <= promptLabelMaxLen {
		return s
	}

	return s[:promptLabelMaxLen] + ellipsis
}

func truncateInlineWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}

	if lipgloss.Width(s) <= width {
		return s
	}

	if width <= len(ellipsis) {
		return ellipsis[:width]
	}

	runes := []rune(s)

	limit := width - len(ellipsis)
	if limit > len(runes) {
		limit = len(runes)
	}

	return string(runes[:limit]) + ellipsis
}
