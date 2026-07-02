package cli

import (
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func TestAskPromptErrorWithAskPayload(t *testing.T) {
	err := askPromptError(session.Event{
		Ask: &ask.Request{Title: "Allow bash?"},
	})
	if err == nil {
		t.Fatal("askPromptError = nil, want error")
	}

	for _, want := range []string{"permission required", "Allow bash?", "non-interactive", "-mode yolo"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to mention %q", err, want)
		}
	}
}

func TestAskPromptErrorWithoutAskPayload(t *testing.T) {
	err := askPromptError(session.Event{})
	if err == nil {
		t.Fatal("askPromptError = nil, want error")
	}

	for _, want := range []string{"permission required", "non-interactive", "-mode yolo"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to mention %q", err, want)
		}
	}
}
