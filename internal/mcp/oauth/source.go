package oauth

import (
	"context"
	"errors"
	"fmt"
	"sync"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"golang.org/x/oauth2"
)

// persistingSource refreshes the access token from the stored refresh token and
// writes the refreshed token back to the store whenever it changes
type persistingSource struct {
	base oauth2.TokenSource

	key     Key
	store   Store
	session Session

	mu      sync.Mutex
	current string
}

func newPersistingSource(session Session, key Key, store Store) *persistingSource {
	cfg := session.oauthConfig("")

	return &persistingSource{
		base:    cfg.TokenSource(context.Background(), session.oauthToken()),
		key:     key,
		store:   store,
		session: session,
	}
}

func (s *persistingSource) Token() (*oauth2.Token, error) {
	token, err := s.base.Token()
	if err != nil {
		if isInvalidGrant(err) {
			return nil, s.invalidate(err)
		}

		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if token.AccessToken != s.current {
		s.current = token.AccessToken
		_ = s.store.Save(context.Background(), s.key, withToken(s.session, token))
	}

	return token, nil
}

func (s *persistingSource) invalidate(err error) error {
	deleteErr := s.store.Delete(context.Background(), s.key)
	if deleteErr != nil {
		return fmt.Errorf("mcp: oauth grant is invalid and clearing saved session failed: %w", errors.Join(mcpauth.ErrOAuth, err, deleteErr))
	}

	return fmt.Errorf("mcp: oauth grant is invalid; re-authentication required: %w", errors.Join(mcpauth.ErrOAuth, err))
}

func isInvalidGrant(err error) bool {
	var retrieveErr *oauth2.RetrieveError
	if !errors.As(err, &retrieveErr) {
		return false
	}

	return retrieveErr.ErrorCode == "invalid_grant"
}
