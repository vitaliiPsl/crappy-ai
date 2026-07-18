package oauth

import (
	"context"
	"errors"
	"sync"
	"time"
)

const refreshMargin = 5 * time.Minute

type source struct {
	mu sync.Mutex

	providerID string
	provider   Provider
	store      Store
}

func newSource(providerID string, provider Provider, store Store) *source {
	return &source{
		providerID: providerID,
		provider:   provider,
		store:      store,
	}
}

func (s *source) resolve(ctx context.Context) (Authorization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	credential, err := s.store.Load(ctx, s.providerID)
	if err != nil {
		return Authorization{}, err
	}

	if credential == nil || credential.AccessToken == "" {
		return Authorization{}, ErrAuthRequired
	}

	if credential.ExpiresAt.IsZero() || time.Now().Add(refreshMargin).Before(credential.ExpiresAt) {
		return s.provider.Authorization(*credential), nil
	}

	refreshed, err := s.provider.Refresh(ctx, *credential)
	if err != nil {
		if errors.Is(err, ErrInvalidGrant) {
			if deleteErr := s.store.Delete(ctx, s.providerID); deleteErr != nil {
				return Authorization{}, errors.Join(ErrAuthRequired, err, deleteErr)
			}

			return Authorization{}, errors.Join(ErrAuthRequired, err)
		}

		return Authorization{}, err
	}

	if refreshed.AccessToken == "" {
		return Authorization{}, errors.New("provider oauth: refresh returned an empty access token")
	}

	if err := s.store.Save(ctx, s.providerID, refreshed); err != nil {
		return Authorization{}, err
	}

	return s.provider.Authorization(refreshed), nil
}
