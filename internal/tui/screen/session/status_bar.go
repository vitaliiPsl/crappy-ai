package session

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/utils"
)

const (
	hintsText           = "Enter Submit • Ctrl+p Sessions • Ctrl+o Thinking • Ctrl+t Tools"
	activeTurnHintsText = "Esc Cancel • Ctrl+p Sessions • Ctrl+o Thinking • Ctrl+t Tools"
)

type statusBar struct {
	turnActive bool
	model      string
	cwd        string
	stats      *sessiondata.TurnStats

	width int
}

func newStatusBar(model, cwd string) statusBar {
	return statusBar{
		model: model,
		cwd:   utils.CompactHome(cwd),
	}
}

func (s statusBar) Update(msg tea.Msg) (statusBar, tea.Cmd) {
	switch msg := msg.(type) {
	case sessionEventMsg:
		switch msg.event.Type {
		case sessiondata.EventTurnComplete:
			s.turnActive = false
			s.stats = msg.event.Stats
		case sessiondata.EventTurnCancelled, sessiondata.EventError:
			s.turnActive = false
		}

	case turnStartedMsg:
		s.turnActive = true

	case turnStoppedMsg:
		s.turnActive = false
	}

	return s, nil
}

func (s statusBar) View() string {
	var parts []string
	if meta := s.metaView(); meta != "" {
		parts = append(parts, meta)
	}

	if hints := s.hintsView(); hints != "" {
		parts = append(parts, hints)
	}

	return strings.Join(parts, "\n")
}

func (s *statusBar) setSize(width int) {
	s.width = width
}

func (s statusBar) metaView() string {
	if s.width <= 0 {
		return ""
	}

	segWidth := max(s.width/3, 1)
	left := truncateLeft(s.cwd, segWidth)
	center := statsText(s.stats)
	right := truncateLeft(s.model, segWidth)

	if left == "" && center == "" && right == "" {
		return ""
	}

	row := []rune(strings.Repeat(" ", s.width))
	placeSegment(row, left, 0)
	placeSegment(row, center, max((s.width-lipgloss.Width(center))/2, 0))
	placeSegment(row, right, max(s.width-lipgloss.Width(right), 0))

	return textStyle.Render(strings.TrimRight(string(row), " "))
}

func (s statusBar) hintsView() string {
	hints := hintsText
	if s.turnActive {
		hints = activeTurnHintsText
	}

	return hintsStyle.Width(s.width).Align(lipgloss.Center).Render(hints)
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
