package oauth

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestHandlerPassiveAuthorizeReturnsAuthorizationRequired(t *testing.T) {
	oauthHandler, err := NewPassiveHandler(HandlerConfig{
		Config: &Config{},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	err = oauthHandler.Authorize(context.Background(), nil, &http.Response{Body: http.NoBody})
	if !errors.Is(err, ErrAuthorizationRequired) {
		t.Fatalf("Authorize() error = %v, want ErrAuthorizationRequired", err)
	}
}

func TestHandlerTokenSourceUsesAuthorizerInPassiveMode(t *testing.T) {
	oauthHandler, err := NewPassiveHandler(HandlerConfig{
		Config: &Config{},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	source, err := oauthHandler.TokenSource(context.Background())
	if err != nil {
		t.Fatalf("TokenSource() error = %v", err)
	}

	if source != nil {
		t.Fatal("TokenSource() = non-nil before authorization, want nil")
	}
}

func TestHandlerInteractiveAuthorizationIsConfigured(t *testing.T) {
	oauthHandler, err := NewInteractiveHandler(HandlerConfig{
		Config: &Config{},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	appHandler, ok := oauthHandler.(*handler)
	if !ok {
		t.Fatalf("handler = %T, want *handler", oauthHandler)
	}

	if _, ok := appHandler.authorization.(interactiveAuthorization); !ok {
		t.Fatalf("authorization = %T, want interactiveAuthorization", appHandler.authorization)
	}
}

func TestHandlerTokenSourceLoadsStoredSession(t *testing.T) {
	key := testSessionKey()

	store := newMemorySessionStore()
	if err := store.Save(context.Background(), key, Session{
		ServerURL: key.ServerURL,
		Token: Token{
			AccessToken: "stored",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	handler := &handler{
		authorizer:   &fakeAuthorizer{},
		sessionKey:   key,
		sessionStore: store,
	}

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

func TestHandlerTokenSourceIgnoresExpiredStoredSession(t *testing.T) {
	key := testSessionKey()

	store := newMemorySessionStore()
	if err := store.Save(context.Background(), key, Session{
		ServerURL: key.ServerURL,
		Token: Token{
			AccessToken: "expired",
			ExpiresAt:   time.Now().Add(-time.Hour),
		},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	handler := &handler{
		authorizer:   &fakeAuthorizer{},
		sessionKey:   key,
		sessionStore: store,
	}

	source, err := handler.TokenSource(context.Background())
	if err != nil {
		t.Fatalf("TokenSource() error = %v", err)
	}

	if source != nil {
		t.Fatal("TokenSource() returned expired token source, want nil")
	}
}

func TestHandlerAuthorizeSavesAuthorizerToken(t *testing.T) {
	key := testSessionKey()
	store := newMemorySessionStore()
	authorizer := &fakeAuthorizer{
		token: &oauth2.Token{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(time.Hour),
		},
	}
	handler := &handler{
		authorizer:    authorizer,
		authorization: interactiveAuthorization{},
		sessionKey:    key,
		sessionStore:  store,
	}

	req, err := http.NewRequest(http.MethodGet, key.ServerURL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	if err := handler.Authorize(context.Background(), req, &http.Response{Body: http.NoBody}); err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}

	if !authorizer.authorized {
		t.Fatal("authorizer Authorize() was not called")
	}

	session, err := store.Load(context.Background(), key)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if session == nil {
		t.Fatal("stored session = nil, want saved token")
	}

	if session.Token.AccessToken != "new-access" || session.Token.RefreshToken != "new-refresh" {
		t.Fatalf("stored token = %+v, want new access/refresh", session.Token)
	}
}

func TestHandlerTokenSourceSavesAuthorizerToken(t *testing.T) {
	key := testSessionKey()
	store := newMemorySessionStore()
	handler := &handler{
		authorizer: &fakeAuthorizer{
			token: &oauth2.Token{
				AccessToken: "live",
				Expiry:      time.Now().Add(time.Hour),
			},
		},
		sessionKey:   key,
		sessionStore: store,
	}

	source, err := handler.TokenSource(context.Background())
	if err != nil {
		t.Fatalf("TokenSource() error = %v", err)
	}

	if _, err := source.Token(); err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	session, err := store.Load(context.Background(), key)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if session == nil || session.Token.AccessToken != "live" {
		t.Fatalf("stored session = %+v, want live token", session)
	}
}

type fakeAuthorizer struct {
	token      *oauth2.Token
	authorized bool
}

func (a *fakeAuthorizer) TokenSource(context.Context) (oauth2.TokenSource, error) {
	if a.token == nil {
		return nil, nil
	}

	return oauth2.StaticTokenSource(a.token), nil
}

func (a *fakeAuthorizer) Authorize(context.Context, *http.Request, *http.Response) error {
	a.authorized = true

	return nil
}

type memorySessionStore struct {
	mu       sync.Mutex
	sessions map[string]Session
}

func newMemorySessionStore() *memorySessionStore {
	return &memorySessionStore{
		sessions: make(map[string]Session),
	}
}

func (s *memorySessionStore) Load(_ context.Context, key SessionKey) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[key.ID()]
	if !ok {
		return nil, nil
	}

	return &session, nil
}

func (s *memorySessionStore) Save(_ context.Context, key SessionKey, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[key.ID()] = session

	return nil
}

func (s *memorySessionStore) Delete(_ context.Context, key SessionKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, key.ID())

	return nil
}

func testSessionKey() SessionKey {
	return NewSessionKey("test", "https://example.com/mcp")
}
