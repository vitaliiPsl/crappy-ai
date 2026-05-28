package session

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	permissionmodel "github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
)

func TestUpdateRoutesPasteToInput(t *testing.T) {
	m := Model{
		state: State{Phase: PhaseIdle},
		input: newInputBar(command.NewRegistry(nil)),
	}

	got, _ := m.Update(tea.PasteMsg{Content: "hello\nfrom paste"})

	if value := got.input.input.Value(); value != "hello\nfrom paste" {
		t.Fatalf("input value = %q, want pasted text", value)
	}
}

func TestUpdateDoesNotPasteIntoInputDuringPermissionPrompt(t *testing.T) {
	m := Model{
		state: State{
			Phase:  PhaseAwaitingPermission,
			Prompt: &permissionmodel.AskRequest{},
		},
		input: newInputBar(command.NewRegistry(nil)),
	}

	got, _ := m.Update(tea.PasteMsg{Content: "ignored"})

	if value := got.input.input.Value(); value != "" {
		t.Fatalf("input value = %q, want empty while permission prompt has focus", value)
	}
}
