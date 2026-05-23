package session

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	emptyLogo = "" +
		"  ██████╗██████╗  █████╗ ██████╗ ██████╗ ██╗   ██╗\n" +
		" ██╔════╝██╔══██╗██╔══██╗██╔══██╗██╔══██╗╚██╗ ██╔╝\n" +
		" ██║     ██████╔╝███████║██████╔╝██████╔╝ ╚████╔╝ \n" +
		" ██║     ██╔══██╗██╔══██║██╔═══╝ ██╔═══╝   ╚██╔╝  \n" +
		" ╚██████╗██║  ██║██║  ██║██║     ██║        ██║   \n" +
		"  ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝        ╚═╝   "
	emptyLogoCompact = "CRAPPY"

	emptyHeadline = "What are we working on today?"
	emptySubtitle = "Notice patterns, untangle thoughts, or decide what matters next."

	emptyPadFactor = 4
)

func renderEmpty(s *State, width, height int) string {
	var b strings.Builder

	b.WriteString(textStyle.Render(chooseLogo(width)))
	b.WriteString("\n\n")
	b.WriteString(textStyle.Render(emptyHeadline))
	b.WriteString("\n")
	b.WriteString(subtleTextStyle.Render(emptySubtitle))
	b.WriteString("\n\n")

	if line := catalogLine(s); line != "" {
		b.WriteString(line)
		b.WriteString("\n")
	}

	content := b.String()
	pad := max((height-strings.Count(content, "\n")-emptyPadFactor)/2, 0)

	return strings.Repeat("\n", pad) + emptyCenterStyle.Width(width).Render(content)
}

func chooseLogo(width int) string {
	if width > 0 && lipgloss.Width(emptyLogo) > width {
		return emptyLogoCompact
	}

	return emptyLogo
}

func catalogLine(s *State) string {
	switch {
	case s.Model != "":
		return subtleTextStyle.Render("Model: ") + textStyle.Render(s.Model)
	case s.Provider != "":
		return subtleTextStyle.Render("Provider: ") + textStyle.Render(s.Provider)
	}

	return ""
}
