package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	Primary   color.Color
	Secondary color.Color
	Muted     color.Color

	Error   color.Color
	Success color.Color
	Warning color.Color

	Text       color.Color
	SubtleText color.Color
	Surface    color.Color
	SurfaceAlt color.Color
}

var Default = Theme{
	Primary:   lipgloss.Color("#89DDFF"),
	Secondary: lipgloss.Color("#f7e76c"),
	Muted:     lipgloss.Color("#c0c0c0"),

	Error:   lipgloss.Color("#FF5370"),
	Success: lipgloss.Color("#C3E88D"),
	Warning: lipgloss.Color("#FFCB6B"),

	Text:       lipgloss.Color("#EEFFFF"),
	SubtleText: lipgloss.Color("#b0b0b0"),
	Surface:    lipgloss.Color("#1E1E2E"),
	SurfaceAlt: lipgloss.Color("#4E4F53"),
}
