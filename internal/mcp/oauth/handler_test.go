package oauth

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestPassiveAuthorizeReturnsAuthorizationRequired(t *testing.T) {
	handler := New(HandlerConfig{Store: newMemoryStore()})

	err := handler.Authorize(context.Background(), nil, &http.Response{Body: http.NoBody})
	if !errors.Is(err, ErrAuthorizationRequired) {
		t.Fatalf("Authorize() error = %v, want ErrAuthorizationRequired", err)
	}
}

func TestTokenSourceNilWithoutStoredSession(t *testing.T) {
	handler := New(HandlerConfig{Key: testKey(), Store: newMemoryStore()})

	source, err := handler.TokenSource(context.Background())
	if err != nil {
		t.Fatalf("TokenSource() error = %v", err)
	}

	if source != nil {
		t.Fatal("TokenSource() = non-nil without a stored session, want nil")
	}
}

func TestTokenSourceUsesStoredSession(t *testing.T) {
	key := testKey()

	store := newMemoryStore()
	if err := store.Save(context.Background(), key, Session{
		ServerURL: key.ServerURL,
		ClientID:  "client",
		AuthURL:   "https://auth.example.com/authorize",
		TokenURL:  "https://auth.example.com/token",
		Token: Token{
			AccessToken: "stored",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	handler := New(HandlerConfig{Key: key, Store: store})

	source, err := handler.TokenSource(context.Background())
	if err != nil {
		t.Fatalf("TokenSource() error = %v", err)
	}

	if source == nil {
		t.Fatal("TokenSource() = nil, want stored token source")
	}

	token, err := source.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	if token.AccessToken != "stored" {
		t.Fatalf("AccessToken = %q, want stored", token.AccessToken)
	}
}

func testKey() Key {
	return NewKey("test", "https://example.com/mcp")
}

type memoryStore struct {
	mu       sync.Mutex
	sessions map[string]Session
}

func newMemoryStore() *memoryStore {
	return &memoryStore{sessions: make(map[string]Session)}
}

func (s *memoryStore) Load(_ context.Context, key Key) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[key.ID()]
	if !ok {
		return nil, nil
	}

	return &session, nil
}

func (s *memoryStore) Save(_ context.Context, key Key, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[key.ID()] = session

	return nil
}

func (s *memoryStore) Delete(_ context.Context, key Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, key.ID())

	return nil
}
