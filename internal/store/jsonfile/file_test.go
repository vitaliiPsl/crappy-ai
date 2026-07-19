package jsonfile

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFileSavesAndLoadsEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "values.json")

	file, err := New[string](path)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := file.Save(context.Background(), "key", "value"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := file.Load(context.Background(), "key")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got == nil || *got != "value" {
		t.Fatalf("Load() = %v, want value", got)
	}
}
