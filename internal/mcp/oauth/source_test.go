package oauth

import (
	"context"
	"errors"
	"net/http"
	"testing"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"golang.org/x/oauth2"
)

func TestPersistingSourceInvalidGrantDeletesSessionAndRequiresAuth(t *testing.T) {
	key := testKey()

	store := newMemoryStore()
	if err := store.Save(context.Background(), key, Session{Token: Token{AccessToken: "old", RefreshToken: "refresh"}}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	source := &persistingSource{
		base: errSource{err: &oauth2.RetrieveError{
			Response:         &http.Response{Status: "400 Bad Request"},
			ErrorCode:        "invalid_grant",
			ErrorDescription: "Grant not found",
		}},
		key:   key,
		store: store,
	}

	_, err := source.Token()
	if !errors.Is(err, mcpauth.ErrOAuth) {
		t.Fatalf("Token() error = %v, want oauth required", err)
	}

	session, loadErr := store.Load(context.Background(), key)
	if loadErr != nil {
		t.Fatalf("Load() error = %v", loadErr)
	}

	if session != nil {
		t.Fatalf("Load() after invalid grant = %+v, want nil", session)
	}
}

func TestPersistingSourceNonInvalidGrantKeepsSession(t *testing.T) {
	key := testKey()

	store := newMemoryStore()
	if err := store.Save(context.Background(), key, Session{Token: Token{AccessToken: "old", RefreshToken: "refresh"}}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	wantErr := errors.New("network down")
	source := &persistingSource{
		base:  errSource{err: wantErr},
		key:   key,
		store: store,
	}

	_, err := source.Token()
	if !errors.Is(err, wantErr) {
		t.Fatalf("Token() error = %v, want %v", err, wantErr)
	}

	session, loadErr := store.Load(context.Background(), key)
	if loadErr != nil {
		t.Fatalf("Load() error = %v", loadErr)
	}

	if session == nil {
		t.Fatal("Load() after non-invalid-grant error = nil, want session preserved")
	}
}

type errSource struct {
	err error
}

func (s errSource) Token() (*oauth2.Token, error) {
	return nil, s.err
}
