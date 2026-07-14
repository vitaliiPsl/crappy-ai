package session

import (
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

var (
	sessionTheme = theme.Default

	textStyle       = lipgloss.NewStyle().Foreground(sessionTheme.Text)
	subtleTextStyle = lipgloss.NewStyle().Foreground(sessionTheme.SubtleText)
	errorStyle      = lipgloss.NewStyle().Foreground(sessionTheme.Error)
	hintsStyle      = lipgloss.NewStyle().Foreground(sessionTheme.SubtleText)
	systemStyle     = lipgloss.NewStyle().Foreground(sessionTheme.Muted)
	thinkingStyle   = lipgloss.NewStyle().Foreground(sessionTheme.Muted)

	thinkingHeaderStyle = lipgloss.NewStyle().
				Foreground(sessionTheme.Muted).
				Italic(true).
				Bold(true)

	toolNameStyle = lipgloss.NewStyle().Foreground(sessionTheme.Text).Bold(true)

	toolBlockBase = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			PaddingLeft(1)
	toolBlockPending = toolBlockBase.BorderForeground(sessionTheme.Warning)
	toolBlockDone    = toolBlockBase.BorderForeground(sessionTheme.Success)
	toolBlockError   = toolBlockBase.BorderForeground(sessionTheme.Error)

	userMessageStyle = lipgloss.NewStyle().
				Foreground(sessionTheme.Text).
				Background(sessionTheme.SurfaceAlt).
				Padding(0, 1)
	queuedMessageStyle = userMessageStyle.
				Foreground(sessionTheme.SubtleText).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(sessionTheme.Warning)

	assistantMessageStyle = lipgloss.NewStyle().Padding(0, 1)

	runIndicatorStyle = lipgloss.NewStyle().Foreground(sessionTheme.SubtleText)
	spinnerStyle      = lipgloss.NewStyle().Foreground(sessionTheme.Primary)
	yoloModeStyle     = lipgloss.NewStyle().Foreground(sessionTheme.Warning).Bold(true)

	promptBoxStyle = lipgloss.NewStyle().
			Background(sessionTheme.SurfaceAlt).
			Padding(0, 1)
	promptPrefixStyle = lipgloss.NewStyle().
				Foreground(sessionTheme.Primary).
				Background(sessionTheme.SurfaceAlt)
	promptQuestionStyle = lipgloss.NewStyle().
				Foreground(sessionTheme.Warning).
				Background(sessionTheme.SurfaceAlt)

	emptyCenterStyle = lipgloss.NewStyle().Align(lipgloss.Center)
)
