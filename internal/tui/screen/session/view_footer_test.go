package session

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestRenderErrorLineIsSingleLineAndFitsWidth(t *testing.T) {
	state := State{LastError: "invalid request:\nGenerateContentRequest.tools[0].name is invalid and much too long"}

	got := renderErrorLine(&state, 40)

	if strings.Contains(got, "\n") {
		t.Fatalf("renderErrorLine() = %q, want one line", got)
	}

	if width := lipgloss.Width(got); width > 40 {
		t.Fatalf("renderErrorLine() width = %d, want <= 40", width)
	}

	if !strings.Contains(got, "Error: invalid request:") {
		t.Fatalf("renderErrorLine() = %q, want error prefix", got)
	}
}
