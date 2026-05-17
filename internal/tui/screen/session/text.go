package session

import (
	"strings"
)

func truncate(text string, maxLen int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)

	if len(text) <= maxLen {
		return text
	}

	return text[:maxLen] + "..."
}
