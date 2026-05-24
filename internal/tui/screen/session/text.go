package session

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const ellipsis = "..."

func truncateInline(text string, maxLen int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)

	if len(text) <= maxLen {
		return text
	}

	return text[:maxLen] + ellipsis
}

func truncateLeft(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if lipgloss.Width(text) <= maxWidth {
		return text
	}

	if maxWidth == 1 {
		return ellipsis
	}

	return ellipsis + text[len(text)-(maxWidth-1):]
}
