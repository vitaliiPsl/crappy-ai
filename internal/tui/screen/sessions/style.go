package sessions

import (
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

var (
	thm = theme.Default

	headerStyle   = lipgloss.NewStyle().Foreground(thm.Primary).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(thm.Primary).Bold(true)
	itemStyle     = lipgloss.NewStyle().Foreground(thm.Text)
	metaStyle     = lipgloss.NewStyle().Foreground(thm.SubtleText)
	emptyStyle    = lipgloss.NewStyle().Foreground(thm.SubtleText)
	errorStyle    = lipgloss.NewStyle().Foreground(thm.Error)
	hintsStyle    = lipgloss.NewStyle().Foreground(thm.SubtleText)
)
