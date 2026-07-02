package session

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
)

const (
	promptPaddingX    = 1
	promptLabelMaxLen = 60
	promptPrompt      = "> "
	promptHintGap     = "g Pattern"
	promptHintGapSep  = ": "
)

const (
	promptOptionAllowOnce    = "allow_once"
	promptOptionAllowExact   = "allow_exact"
	promptOptionAllowPattern = "allow_pattern"
	promptOptionDenyOnce     = "deny_once"
)

func renderPrompt(req *ask.Request, width int) string {
	prefix := promptPrefixStyle.Render(promptPrompt)
	body := promptQuestionStyle.Render(req.Title)
	line := lipgloss.JoinHorizontal(lipgloss.Top, prefix, body)

	return strings.TrimRight(promptBoxStyle.Width(width).Render("\n"+line+"\n"), "\n")
}

func renderPromptHints(req *ask.Request, width int) string {
	hints := []string{"y Once"}

	if _, ok := askOption(*req, promptOptionAllowExact); ok {
		hints = append(hints, "e Exact")
	}

	if option, ok := askOption(*req, promptOptionAllowPattern); ok {
		hints = append(hints, promptPatternHint(option, hints, width))
	}

	hints = append(hints, "n Deny")

	return strings.Join(hints, " • ")
}

func pickPromptOption(key tea.KeyMsg, req ask.Request) string {
	candidate := ""

	switch key.String() {
	case "y", "enter":
		candidate = promptOptionAllowOnce
	case "e":
		candidate = promptOptionAllowExact
	case "g":
		candidate = promptOptionAllowPattern
	case "n", "esc":
		candidate = promptOptionDenyOnce
	}

	if candidate == "" {
		return ""
	}

	if _, ok := askOption(req, candidate); !ok {
		return ""
	}

	return candidate
}

func askOption(req ask.Request, id string) (ask.Option, bool) {
	for _, option := range req.Options {
		if option.ID == id {
			return option, true
		}
	}

	return ask.Option{}, false
}

func promptPatternHint(option ask.Option, previous []string, width int) string {
	if option.Label == "" {
		return promptHintGap
	}

	prefix := promptHintGap + promptHintGapSep
	tail := "n Deny"
	available := width - lipgloss.Width(strings.Join(append(append([]string{}, previous...), prefix, tail), " • "))

	if available <= len(ellipsis) {
		return promptHintGap
	}

	return prefix + truncateInlineWidth(option.Label, available)
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
