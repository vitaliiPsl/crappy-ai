package memory

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	corememory "github.com/vitaliiPsl/crappy-ai/internal/memory"
	memoryStore "github.com/vitaliiPsl/crappy-ai/internal/memory/store"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
)

func TestModelCreatesEditsAndDeletesMemory(t *testing.T) {
	store, err := memoryStore.NewFileStore(filepath.Join(t.TempDir(), "memory.json"))
	if err != nil {
		t.Fatal(err)
	}

	srv := server.New(nil, nil, nil, nil, nil, nil, nil, store)
	model := New(context.Background(), srv)
	model.SetSize(100, 30)

	model, _ = model.startCreating()
	model.draft.Kind = corememory.KindPreference
	model.draft.Content = "Prefers concise answers."
	model = applyLoaded(t, model, model.saveDraft())

	if len(model.memories) != 1 || model.memories[0].Kind != corememory.KindPreference {
		t.Fatalf("created memories = %+v", model.memories)
	}

	if view := model.View(); !strings.Contains(view, "Prefers concise answers.") || !strings.Contains(view, "preference") {
		t.Fatalf("View() missing memory:\n%s", view)
	}

	model, _ = model.startEditing()
	model.draft.Kind = corememory.KindInstruction
	model.draft.Content = "Do not use emojis."
	model = applyLoaded(t, model, model.saveDraft())

	if model.memories[0].Content != "Do not use emojis." {
		t.Fatalf("updated memories = %+v", model.memories)
	}

	model = applyLoaded(t, model, model.deleteMemory(model.selectedID()))
	if len(model.memories) != 0 {
		t.Fatalf("memories after delete = %+v", model.memories)
	}
}

func TestCycleKindWraps(t *testing.T) {
	model := Model{draft: corememory.Memory{Kind: corememory.KindProfile}}

	model.cycleKind()

	if model.draft.Kind != corememory.KindPreference {
		t.Fatalf("first cycle = %q", model.draft.Kind)
	}

	model.cycleKind()
	model.cycleKind()

	if model.draft.Kind != corememory.KindProfile {
		t.Fatalf("wrapped cycle = %q", model.draft.Kind)
	}
}

func applyLoaded(t *testing.T, model Model, cmd tea.Cmd) Model {
	t.Helper()

	msg, ok := cmd().(memoriesLoadedMsg)
	if !ok {
		t.Fatalf("command returned unexpected message")
	}

	updated, _ := model.Update(msg)

	return updated
}
