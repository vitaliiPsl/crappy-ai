package session

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
)

func TestUpdateRoutesPasteToInput(t *testing.T) {
	m := Model{
		state: State{Phase: PhaseIdle},
		input: newInputBar(command.NewRegistry(context.Background())),
	}

	got, _ := m.Update(tea.PasteMsg{Content: "hello\nfrom paste"})

	if value := got.input.input.Value(); value != "hello\nfrom paste" {
		t.Fatalf("input value = %q, want pasted text", value)
	}
}

func TestUpdateDoesNotPasteIntoInputDuringPrompt(t *testing.T) {
	m := Model{
		state: State{
			Phase:  PhaseAwaitingPermission,
			Prompt: &ask.Request{},
		},
		input: newInputBar(command.NewRegistry(context.Background())),
	}

	got, _ := m.Update(tea.PasteMsg{Content: "ignored"})

	if value := got.input.input.Value(); value != "" {
		t.Fatalf("input value = %q, want empty while prompt has focus", value)
	}
}
