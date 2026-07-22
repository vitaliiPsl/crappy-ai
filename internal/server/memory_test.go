package server

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/memory"
	memoryStore "github.com/vitaliiPsl/crappy-ai/internal/memory/store"
)

func TestMemoryLifecycle(t *testing.T) {
	store, err := memoryStore.NewFileStore(filepath.Join(t.TempDir(), "memory.json"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	srv := New(nil, nil, nil, nil, nil, nil, nil, store)

	created, err := srv.CreateMemory(context.Background(), memory.CreateParams{
		Kind:    memory.KindPreference,
		Content: "Prefers concise answers.",
	})
	if err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}

	updated, err := srv.UpdateMemory(context.Background(), memory.UpdateParams{
		ID:      created.ID,
		Kind:    memory.KindInstruction,
		Content: "Do not use emojis.",
	})
	if err != nil {
		t.Fatalf("UpdateMemory: %v", err)
	}

	if updated.Kind != memory.KindInstruction {
		t.Fatalf("updated memory = %+v", updated)
	}

	memories, err := srv.ListMemories(context.Background())
	if err != nil || len(memories) != 1 {
		t.Fatalf("ListMemories = %+v, %v", memories, err)
	}

	if err := srv.DeleteMemory(context.Background(), created.ID); err != nil {
		t.Fatalf("DeleteMemory: %v", err)
	}
}
