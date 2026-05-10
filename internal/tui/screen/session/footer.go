package session

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/component"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

const (
	hintsText          = "Enter Submit • Shift+Enter New Line • Ctrl+o Thinking • Ctrl+t Tools"
	streamingHintsText = "Esc Cancel • Ctrl+o Thinking • Ctrl+t Tools"
	streamingLabel     = "Thinking..."
)

type footer struct {
	input   component.Input
	spinner spinner.Model

	streaming bool
	model     string
	stats     *sessiondata.TurnStats

	width int
}

func newFooter(model string) footer {
	thm := theme.Default
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(thm.Primary)

	return footer{
		input:   component.NewInput(),
		spinner: sp,
		model:   model,
	}
}

func (f footer) Init() tea.Cmd {
	return f.input.Init()
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

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+o", "ctrl+t", "pgup", "pgdown", "up", "down":
			return f, nil, false
		}
	}

	if _, ok := msg.(tea.MouseWheelMsg); ok {
		return f, nil, false
	}

	switch msg.(type) {
	case tea.KeyMsg, tea.PasteMsg, tea.PasteStartMsg, tea.PasteEndMsg:
	default:
		return f, nil, false
	}

	var cmd tea.Cmd

	f.input, cmd, _ = f.input.Update(msg)

	return f, cmd, true
}

func (f footer) View() string {
	var parts []string

	if status := f.statusView(); status != "" {
		parts = append(parts, status)
	}

	parts = append(parts, strings.TrimRight(f.input.View(), "\n"))

	if meta := f.metaView(); meta != "" {
		parts = append(parts, meta)
	}

	parts = append(parts, hintsStyle.Width(f.width).Align(lipgloss.Center).Render(f.hintsText()))

	return strings.Join(parts, "\n")
}

func (f footer) Height() int {
	return lipgloss.Height(f.View())
}

func (f *footer) setSize(width int) {
	f.width = width
	f.input.SetWidth(width)
}

func (f footer) statusView() string {
	if !f.streaming {
		return ""
	}

	return subtleTextStyle.Width(f.width).Render(f.spinner.View() + " " + streamingLabel)
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

	left := f.model
	center := statsText(f.stats)

	if left == "" && center == "" {
		return ""
	}

	row := []rune(strings.Repeat(" ", f.width))
	placeSegment(row, truncate(left, max(f.width/3, 1)), 0)
	placeSegment(row, center, max((f.width-lipgloss.Width(center))/2, 0))

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

	return strings.Join(parts, " | ")
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
