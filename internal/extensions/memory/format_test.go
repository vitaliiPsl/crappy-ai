package memory

import (
	"strings"
	"testing"

	corememory "github.com/vitaliiPsl/crappy-ai/internal/memory"
)

func TestFormatContextGroupsKinds(t *testing.T) {
	got := formatContext([]corememory.Memory{
		{Kind: corememory.KindPreference, Content: "Prefers concise answers."},
		{Kind: corememory.KindProfile, Content: "Writes Go."},
		{Kind: corememory.KindInstruction, Content: "Do not use emojis."},
	})

	for _, want := range []string{"## Profile", "- Writes Go.", "## Preferences", "## Instructions"} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatContext() missing %q:\n%s", want, got)
		}
	}
}
