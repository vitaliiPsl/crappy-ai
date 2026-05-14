package session

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

const (
	hintsText           = "Enter Submit • Ctrl+p Sessions • Ctrl+o Thinking • Ctrl+t Tools"
	activeTurnHintsText = "Esc Cancel • Ctrl+p Sessions • Ctrl+o Thinking • Ctrl+t Tools"
	defaultTurnLabel    = "Generating..."
)

type statusBar struct {
	spinner spinner.Model

	turnActive bool
	model      string
	stats      *sessiondata.TurnStats

	width int
}

func newStatusBar(model string) statusBar {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(sessionTheme.Primary)

	return statusBar{
		spinner: sp,
		model:   model,
	}
}

func (s statusBar) Update(msg tea.Msg) (statusBar, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case sessionEventMsg:
		switch msg.event.Type {
		case sessiondata.EventTurnComplete:
			s.turnActive = false
			s.stats = msg.event.Stats
		case sessiondata.EventTurnCancelled, sessiondata.EventError:
			s.turnActive = false
		}

		return s, nil, false

	case turnStartedMsg:
		s.turnActive = true

		return s, s.spinner.Tick, false

	case turnStoppedMsg:
		s.turnActive = false

		return s, nil, false

	case spinner.TickMsg:
		if !s.turnActive {
			return s, nil, true
		}

		var cmd tea.Cmd

		s.spinner, cmd = s.spinner.Update(msg)

		return s, cmd, true
	}

	return s, nil, false
}

func (s statusBar) TurnActive() bool {
	return s.turnActive
}

func (s *statusBar) setSize(width int) {
	s.width = width
}

func (s statusBar) StatusView() string {
	if !s.turnActive {
		return ""
	}

	return subtleTextStyle.Width(s.width).Render(s.spinner.View() + " " + defaultTurnLabel)
}

func (s statusBar) HintsView() string {
	hints := s.hintsText()
	if hints == "" {
		return ""
	}

	return hintsStyle.Width(s.width).Align(lipgloss.Center).Render(hints)
}

func (s statusBar) hintsText() string {
	if s.turnActive {
		return activeTurnHintsText
	}

	return hintsText
}

func (s statusBar) MetaView() string {
	if s.width <= 0 {
		return ""
	}

	center := statsText(s.stats)
	right := truncateLeft(s.model, max(s.width/3, 1))

	if center == "" && right == "" {
		return ""
	}

	row := []rune(strings.Repeat(" ", s.width))
	placeSegment(row, center, max((s.width-lipgloss.Width(center))/2, 0))
	placeSegment(row, right, max(s.width-lipgloss.Width(right), 0))

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
