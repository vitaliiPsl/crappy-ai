package oauth

import (
	"context"

	"golang.org/x/oauth2"
)

type savingTokenSource struct {
	source     oauth2.TokenSource
	sessionKey SessionKey
	store      SessionStore
}

func newSavingTokenSource(source oauth2.TokenSource, sessionKey SessionKey, store SessionStore) oauth2.TokenSource {
	return savingTokenSource{
		source:     source,
		sessionKey: sessionKey,
		store:      store,
	}
}

func (s savingTokenSource) Token() (*oauth2.Token, error) {
	token, err := s.source.Token()
	if err != nil || token == nil || token.AccessToken == "" {
		return token, err
	}

	_ = s.store.Save(context.Background(), s.sessionKey, sessionFromToken(s.sessionKey.ServerURL, token))

	return token, nil
}
