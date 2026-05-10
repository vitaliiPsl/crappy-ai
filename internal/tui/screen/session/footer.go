package session

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/component"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

const (
	hintsText             = "Enter Submit • Ctrl+p Sessions • Ctrl+o Thinking • Ctrl+t Tools"
	streamingHintsText    = "Esc Cancel • Ctrl+p Sessions • Ctrl+o Thinking • Ctrl+t Tools"
	defaultStreamingLabel = "Generating..."

	inputPlaceholder = "Type a message or /command..."
	inputPrompt      = "> "
	inputMaxHeight   = 8
)

type footer struct {
	input       component.Input
	suggestions commandSuggestions
	spinner     spinner.Model

	streaming bool
	model     string
	stats     *sessiondata.TurnStats

	width int
}

func newFooter(registry *command.Registry, model string) footer {
	thm := theme.Default
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(thm.Primary)

	return footer{
		input: component.NewInput(
			component.WithMultiline(true),
			component.WithPlaceholder(inputPlaceholder),
			component.WithPrompt(inputPrompt),
			component.WithMaxHeight(inputMaxHeight),
		),
		suggestions: newCommandSuggestions(registry),
		spinner:     sp,
		model:       model,
	}
}

func (f footer) Init() tea.Cmd {
	return f.input.Focus()
}

func (f footer) Update(msg tea.Msg) (footer, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case sessionEventMsg:
		switch msg.event.Type {
		case sessiondata.EventTurnComplete:
			f.streaming = false
			f.stats = msg.event.Stats
		case sessiondata.EventTurnCancelled, sessiondata.EventError:
			f.streaming = false
		}

		return f, nil, false

	case streamStartedMsg:
		f.streaming = true

		return f, f.spinner.Tick, false

	case turnStoppedMsg:
		f.streaming = false

		return f, nil, false

	case spinner.TickMsg:
		if !f.streaming {
			return f, nil, true
		}

		var cmd tea.Cmd

		f.spinner, cmd = f.spinner.Update(msg)

		return f, cmd, true
	}

	if f.streaming {
		return f, nil, false
	}

	if f.shouldPassThrough(msg) {
		return f, nil, false
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		if next, consumed := f.handleSuggestionKey(key); consumed {
			return next, nil, true
		}
	}

	if !isInputMsg(msg) {
		return f, nil, false
	}

	return f.updateInput(msg)
}

func (f footer) shouldPassThrough(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		return true
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+o", "ctrl+t", "pgup", "pgdown":
			return true
		case "up", "down":
			return !f.suggestions.Active()
		}
	}

	return false
}

func (f footer) handleSuggestionKey(key tea.KeyMsg) (footer, bool) {
	switch key.String() {
	case "up":
		consumed := f.suggestions.Previous()

		return f, consumed
	case "down":
		consumed := f.suggestions.Next()

		return f, consumed
	case "esc":
		if !f.suggestions.Active() {
			return f, false
		}

		f.suggestions.Clear()

		return f, true
	case "enter":
		value, ok := f.suggestions.Completion(f.input.Value())
		if !ok {
			return f, false
		}

		f.input.SetValue(value)
		f.suggestions.Clear()

		return f, true
	default:
		return f, false
	}
}

func isInputMsg(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.KeyMsg, tea.PasteMsg, tea.PasteStartMsg, tea.PasteEndMsg:
		return true
	default:
		return false
	}
}

func (f footer) updateInput(msg tea.Msg) (footer, tea.Cmd, bool) {
	var (
		cmd tea.Cmd
		out tea.Msg
	)

	f.input, cmd, out = f.input.Update(msg)
	f.suggestions.Update(f.input.Value())

	if submit, ok := out.(component.ConfirmMsg); ok {
		return f.handleSubmit(submit.Value)
	}

	return f, cmd, true
}

func (f footer) handleSubmit(value string) (footer, tea.Cmd, bool) {
	if strings.TrimSpace(value) == "" {
		return f, nil, true
	}

	f.input.Reset()
	f.suggestions.Clear()

	if cmdMsg, ok := parseCommand(value); ok {
		return f, func() tea.Msg { return cmdMsg }, true
	}

	return f, func() tea.Msg { return submitMsg{Text: value} }, true
}

func (f footer) View() string {
	var parts []string

	if status := f.statusView(); status != "" {
		parts = append(parts, status)
	}

	if suggestions := f.suggestions.View(); suggestions != "" {
		parts = append(parts, suggestions)
	}

	parts = append(parts, strings.TrimRight(f.bodyView(), "\n"))

	if meta := f.metaView(); meta != "" {
		parts = append(parts, meta)
	}

	if hints := f.hintsView(); hints != "" {
		parts = append(parts, hints)
	}

	return strings.Join(parts, "\n")
}

func (f footer) Height() int {
	return lipgloss.Height(f.View())
}

func (f *footer) setSize(width int) {
	f.width = width
	f.input.SetWidth(width)
}

func (f footer) bodyView() string {
	return f.input.View()
}

func (f footer) statusView() string {
	if !f.streaming {
		return ""
	}

	return subtleTextStyle.Width(f.width).Render(f.spinner.View() + " " + defaultStreamingLabel)
}

func (f footer) hintsView() string {
	hints := f.hintsText()
	if hints == "" {
		return ""
	}

	return hintsStyle.Width(f.width).Align(lipgloss.Center).Render(hints)
}

func (f footer) hintsText() string {
	if f.streaming {
		return streamingHintsText
	}

	return hintsText
}

func (f footer) metaView() string {
	if f.width <= 0 {
		return ""
	}

	center := statsText(f.stats)
	right := truncateLeft(f.model, max(f.width/3, 1))

	if center == "" && right == "" {
		return ""
	}

	row := []rune(strings.Repeat(" ", f.width))
	placeSegment(row, center, max((f.width-lipgloss.Width(center))/2, 0))
	placeSegment(row, right, max(f.width-lipgloss.Width(right), 0))

	return textStyle.Render(strings.TrimRight(string(row), " "))
}

func statsText(stats *sessiondata.TurnStats) string {
	if stats == nil {
		return ""
	}

	parts := []string{
		fmt.Sprintf("%s in", formatTokens(stats.Usage.InputTokens)),
		fmt.Sprintf("%s out", formatTokens(stats.Usage.OutputTokens)),
	}

	if stats.ContextWindow > 0 {
		pct := int(float64(stats.ContextUsed) / float64(stats.ContextWindow) * 100)
		parts = append(parts, fmt.Sprintf("%d%% ctx", pct))
	}

	return strings.Join(parts, " · ")
}

func truncateLeft(text string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	if lipgloss.Width(text) <= maxLen {
		return text
	}

	if maxLen == 1 {
		return titleEllipsis
	}

	return titleEllipsis + text[len(text)-(maxLen-1):]
}

func placeSegment(row []rune, text string, start int) {
	if text == "" || start >= len(row) {
		return
	}

	runes := []rune(text)
	if start < 0 {
		runes = runes[-start:]
		start = 0
	}

	for i, r := range runes {
		pos := start + i
		if pos >= len(row) {
			break
		}

		row[pos] = r
	}
}

func formatTokens(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}

	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}

	return fmt.Sprintf("%d", n)
}
