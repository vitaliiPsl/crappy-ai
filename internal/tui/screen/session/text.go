package session

import (
	"regexp"
	"strings"
)

var blankLinesRe = regexp.MustCompile(`\n{2,}`)

func collapseBlankLines(text string) string {
	text = strings.TrimSpace(text)

	return blankLinesRe.ReplaceAllString(text, "\n")
}

func truncate(text string, maxLen int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)

	if len(text) <= maxLen {
		return text
	}

	return text[:maxLen] + "..."
}
