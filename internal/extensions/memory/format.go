package memory

import (
	"fmt"
	"strings"

	corememory "github.com/vitaliiPsl/crappy-ai/internal/memory"
)

const contextPreamble = `# Saved memories

These are user-approved memories from previous interactions. They may be outdated. Prefer the current user request, user-provided project instructions, and directly observed evidence.`

func formatContext(memories []corememory.Memory) string {
	if len(memories) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(contextPreamble)

	for _, group := range []struct {
		kind  corememory.Kind
		title string
	}{
		{corememory.KindProfile, "Profile"},
		{corememory.KindPreference, "Preferences"},
		{corememory.KindInstruction, "Instructions"},
	} {
		var entries []corememory.Memory
		for _, item := range memories {
			if item.Kind == group.kind {
				entries = append(entries, item)
			}
		}

		if len(entries) == 0 {
			continue
		}

		fmt.Fprintf(&b, "\n\n## %s\n", group.title)

		for _, item := range entries {
			fmt.Fprintf(&b, "\n- %s", item.Content)
		}
	}

	return b.String()
}

func formatList(memories []corememory.Memory) string {
	if len(memories) == 0 {
		return "No persistent memories saved."
	}

	var b strings.Builder
	for i, item := range memories {
		if i > 0 {
			b.WriteByte('\n')
		}

		fmt.Fprintf(&b, "- id: %s\n  kind: %s\n  content: %s\n  created_at: %s\n  updated_at: %s",
			item.ID, item.Kind, item.Content, item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	return b.String()
}
