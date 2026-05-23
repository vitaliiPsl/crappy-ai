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
