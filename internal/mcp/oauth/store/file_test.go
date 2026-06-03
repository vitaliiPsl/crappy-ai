package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth"
)

func TestFileStoreSavesAndLoadsSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth.json")
	key := testKey()
	want := oauth.Session{
		ServerURL: key.ServerURL,
		ClientID:  "client-123",
		AuthURL:   "https://auth.example.com/authorize",
		TokenURL:  "https://auth.example.com/token",
		Scopes:    []string{"read", "write"},
		Token: oauth.Token{
			AccessToken:  "access",
			RefreshToken: "refresh",
			TokenType:    "Bearer",
			ExpiresAt:    time.Now().Add(time.Hour).UTC(),
		},
	}

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	if err := store.Save(context.Background(), key, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	fresh, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() fresh error = %v", err)
	}

	got, err := fresh.Load(context.Background(), key)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got == nil {
		t.Fatal("Load() session = nil, want saved session")
	}

	if got.ClientID != want.ClientID ||
		got.TokenURL != want.TokenURL ||
		got.Token.AccessToken != want.Token.AccessToken ||
		got.Token.RefreshToken != want.Token.RefreshToken ||
		!got.Token.ExpiresAt.Equal(want.Token.ExpiresAt) {
		t.Fatalf("session = %+v, want %+v", got, want)
	}
}

func TestFileStoreUsesPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "oauth.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	key := testKey()
	if err := store.Save(context.Background(), key, oauth.Session{
		ServerURL: key.ServerURL,
		Token:     oauth.Token{AccessToken: "access"},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}

	if mode := dirInfo.Mode().Perm(); mode != 0o700 {
		t.Fatalf("dir mode = %o, want 700", mode)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}

	if mode := fileInfo.Mode().Perm(); mode != 0o600 {
		t.Fatalf("file mode = %o, want 600", mode)
	}
}

func TestFileStoreDeleteRemovesSession(t *testing.T) {
	store, err := NewFileStore(filepath.Join(t.TempDir(), "oauth.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	key := testKey()
	if err := store.Save(context.Background(), key, oauth.Session{
		ServerURL: key.ServerURL,
		Token:     oauth.Token{AccessToken: "access"},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	got, err := store.Load(context.Background(), key)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got != nil {
		t.Fatalf("Load() after Delete = %+v, want nil", got)
	}
}

func testKey() oauth.Key {
	return oauth.NewKey("test", "https://example.com/mcp")
}
