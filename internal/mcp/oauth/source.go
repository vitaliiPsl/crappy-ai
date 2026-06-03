package oauth

import (
	"context"
	"sync"

	"golang.org/x/oauth2"
)

type persistingSource struct {
	base oauth2.TokenSource

	key     Key
	store   Store
	session Session

	mu      sync.Mutex
	current string
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
