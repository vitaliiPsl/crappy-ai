package session

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/utils"
)

const (
	hintsIdle    = "PgUp/PgDn Scroll • Enter Submit • Ctrl+o Thinking • Ctrl+t Tools"
	hintsRunning = "Esc Cancel • PgUp/PgDn Scroll • Ctrl+o Thinking • Ctrl+t Tools"

	runLabelThinking   = "Thinking..."
	runLabelGenerating = "Generating..."
	runLabelWorking    = "Working..."
	runLabelCompacting = "Compacting context..."
)

type FooterOpts struct {
	Width   int
	Spinner string
	Input   string
}

func renderFooter(s *State, opts FooterOpts) string {
	var lines []string

	if line := renderRunLine(s, opts); line != "" {
		lines = append(lines, line)
	}

	if body := renderBody(s, opts); body != "" {
		lines = append(lines, body)
	}

	if line := renderErrorLine(s, opts.Width); line != "" {
		lines = append(lines, line)
	}

	if line := renderMetaRow(s, opts.Width); line != "" {
		lines = append(lines, line)
	}

	lines = append(lines, renderHints(s, opts.Width))

	return strings.Join(lines, "\n")
}

func renderBody(s *State, opts FooterOpts) string {
	if s.Prompt != nil {
		return renderPrompt(s.Prompt, opts.Width)
	}

	return opts.Input
}

func renderRunLine(s *State, opts FooterOpts) string {
	if s.Phase != PhaseRunning && s.Phase != PhaseCompacting {
		return ""
	}

	label := runLabel(s)

	return runIndicatorStyle.Width(opts.Width).Render(opts.Spinner + " " + label)
}

func runLabel(s *State) string {
	if active := s.ActiveTool(); active != nil {
		return "Running " + activeToolLabel(active)
	}

	switch s.Activity {
	case ActivityThinking:
		return runLabelThinking
	case ActivityGenerating:
		return runLabelGenerating
	case ActivityCompacting:
		return runLabelCompacting
	default:
		return runLabelWorking
	}
}

func renderErrorLine(s *State, width int) string {
	if s.LastError == "" {
		return ""
	}

	return errorStyle.Width(width).Render("Error: " + s.LastError)
}

func renderMetaRow(s *State, width int) string {
	if width <= 0 {
		return ""
	}

	segWidth := max(width/3, 1)
	left := truncateLeft(utils.CompactHome(s.Cwd), segWidth)
	center := statsLabel(s.Stats)
	right := truncateLeft(s.Model, segWidth)

	if left == "" && center == "" && right == "" {
		return ""
	}

	row := []rune(strings.Repeat(" ", width))
	placeSegment(row, left, 0)
	placeSegment(row, center, max((width-lipgloss.Width(center))/2, 0))
	placeSegment(row, right, max(width-lipgloss.Width(right), 0))

	return subtleTextStyle.Render(strings.TrimRight(string(row), " "))
}

func renderHints(s *State, width int) string {
	hints := pickHints(s, width)

	return hintsStyle.Width(width).Align(lipgloss.Center).Render(hints)
}

func pickHints(s *State, width int) string {
	if s.Prompt != nil {
		return renderPromptHints(s.Prompt, width)
	}

	if s.Phase == PhaseRunning {
		return hintsRunning
	}

	return hintsIdle
}

func activeToolLabel(tool *ToolUse) string {
	if arg := toolInlineArg(tool); arg != "" {
		return fmt.Sprintf("%s %s", tool.Name, arg)
	}

	return tool.Name
}

func statsLabel(stats *sessiondata.TurnStats) string {
	if stats == nil {
		return ""
	}

	parts := []string{
		fmt.Sprintf("%s in", utils.FormatTokens(stats.Usage.InputTokens)),
		fmt.Sprintf("%s out", utils.FormatTokens(stats.Usage.OutputTokens)),
	}

	if stats.ContextWindow > 0 {
		pct := int(float64(stats.ContextUsed) / float64(stats.ContextWindow) * 100)
		parts = append(parts, fmt.Sprintf("%d%% ctx", pct))
	}

	return strings.Join(parts, " · ")
}
