package session

import (
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

var (
	sessionTheme = theme.Default

	thinkingLabelStyle = lipgloss.NewStyle().Foreground(sessionTheme.Muted).Italic(true)

	textStyle       = lipgloss.NewStyle().Foreground(sessionTheme.Text)
	subtleTextStyle = lipgloss.NewStyle().Foreground(sessionTheme.SubtleText)
	thinkingStyle   = lipgloss.NewStyle().Foreground(sessionTheme.Muted)
	errorStyle      = lipgloss.NewStyle().Foreground(sessionTheme.Error)
	hintsStyle      = lipgloss.NewStyle().Foreground(sessionTheme.SubtleText)
	systemStyle     = lipgloss.NewStyle().Foreground(sessionTheme.Muted)
	successStyle    = lipgloss.NewStyle().Foreground(sessionTheme.Success)
	warningStyle    = lipgloss.NewStyle().Foreground(sessionTheme.Warning)

	userMessageStyle = lipgloss.NewStyle().
				Background(sessionTheme.SurfaceAlt).
				Padding(0, 1)

	assistantMessageStyle = lipgloss.NewStyle().
				Padding(0, 1)

	emptyCenterStyle = lipgloss.NewStyle().Align(lipgloss.Center)
)
