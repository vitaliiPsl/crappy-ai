package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vitaliiPsl/crappy-ai/internal/memory"
)

func TestFileStoreLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	created, err := store.Create(context.Background(), memory.CreateParams{
		Kind:    memory.KindPreference,
		Content: " Prefers concise answers. ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID == "" || created.Content != "Prefers concise answers." || created.CreatedAt != now {
		t.Fatalf("Create() = %+v", created)
	}

	store.now = func() time.Time { return now.Add(time.Hour) }

	updated, err := store.Update(context.Background(), memory.UpdateParams{
		ID:      created.ID,
		Kind:    memory.KindInstruction,
		Content: "Do not use emojis.",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.CreatedAt != now || updated.UpdatedAt != now.Add(time.Hour) {
		t.Fatalf("Update() timestamps = %+v", updated)
	}

	memories, err := store.List(context.Background())
	if err != nil || len(memories) != 1 || memories[0].Kind != memory.KindInstruction {
		t.Fatalf("List() = %+v, %v", memories, err)
	}

	if err := store.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	memories, err = store.List(context.Background())
	if err != nil || len(memories) != 0 {
		t.Fatalf("List() after delete = %+v, %v", memories, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("memory mode = %o, want 600", info.Mode().Perm())
	}
}

func TestFileStoreRejectsInvalidAndDuplicateMemories(t *testing.T) {
	store, err := NewFileStore(filepath.Join(t.TempDir(), "memory.json"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Create(context.Background(), memory.CreateParams{Kind: "other", Content: "Something"}); err == nil {
		t.Fatal("Create() accepted invalid kind")
	}

	if _, err := store.Create(context.Background(), memory.CreateParams{Kind: memory.KindProfile, Content: "   "}); err == nil {
		t.Fatal("Create() accepted empty content")
	}

	if _, err := store.Create(context.Background(), memory.CreateParams{Kind: memory.KindProfile, Content: "Writes Go"}); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Create(context.Background(), memory.CreateParams{Kind: memory.KindProfile, Content: "  writes   go  "}); err == nil {
		t.Fatal("Create() accepted normalized duplicate")
	}
}
