package memory

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"

	corememory "github.com/vitaliiPsl/crappy-ai/internal/memory"
)

type fakeStore struct {
	memories []corememory.Memory
}

func (s *fakeStore) List(context.Context) ([]corememory.Memory, error) {
	return append([]corememory.Memory(nil), s.memories...), nil
}

func (s *fakeStore) Create(_ context.Context, params corememory.CreateParams) (corememory.Memory, error) {
	created := corememory.Memory{
		ID: "memory-1", Kind: params.Kind, Content: params.Content, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	s.memories = append(s.memories, created)

	return created, nil
}

func (s *fakeStore) Update(_ context.Context, params corememory.UpdateParams) (corememory.Memory, error) {
	for i := range s.memories {
		if s.memories[i].ID == params.ID {
			s.memories[i].Kind = params.Kind
			s.memories[i].Content = params.Content

			return s.memories[i], nil
		}
	}

	return corememory.Memory{}, fmt.Errorf("not found")
}

func (s *fakeStore) Delete(_ context.Context, id string) error {
	for i := range s.memories {
		if s.memories[i].ID == id {
			s.memories = append(s.memories[:i], s.memories[i+1:]...)

			return nil
		}
	}

	return fmt.Errorf("not found")
}

func TestMemoryToolsLifecycle(t *testing.T) {
	store := &fakeStore{}
	tools := newTools(store)
	rc := kit.NewRunContext(context.Background())

	execute := func(name string, args map[string]any) string {
		t.Helper()

		for _, candidate := range tools {
			if candidate.Definition().Name != name {
				continue
			}

			out, err := candidate.Execute(rc, kit.NewToolCall("call-1", name, args))
			if err != nil {
				t.Fatalf("%s Execute() error = %v", name, err)
			}

			return kit.ContentsText(out.Content)
		}

		t.Fatalf("tool %q not found", name)

		return ""
	}

	if got := execute(toolRemember, map[string]any{"kind": "preference", "content": "Prefers concise answers."}); !strings.Contains(got, "memory-1") {
		t.Fatalf("remember output = %q", got)
	}

	if got := execute(toolList, map[string]any{}); !strings.Contains(got, "Prefers concise answers.") {
		t.Fatalf("list output = %q", got)
	}

	execute(toolUpdate, map[string]any{"id": "memory-1", "kind": "instruction", "content": "Do not use emojis."})

	if store.memories[0].Kind != corememory.KindInstruction {
		t.Fatalf("updated memory = %+v", store.memories[0])
	}

	execute(toolForget, map[string]any{"id": "memory-1"})

	if len(store.memories) != 0 {
		t.Fatalf("memories after forget = %+v", store.memories)
	}
}
