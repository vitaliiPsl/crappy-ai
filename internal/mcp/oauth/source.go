package oauth

import (
	"context"
	"sync"

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
