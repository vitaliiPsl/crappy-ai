package store

import (
	"context"
	"path/filepath"
	"testing"

	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

func TestFileStorePersistsCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	want := provideroauth.Credential{AccessToken: "access", RefreshToken: "refresh"}
	if err := store.Save(context.Background(), "openai", want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	fresh, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() fresh error = %v", err)
	}

	got, err := fresh.Load(context.Background(), "openai")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got == nil || got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
}
