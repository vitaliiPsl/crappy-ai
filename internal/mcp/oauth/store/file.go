package store

import (
	"context"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth"
	oauthstore "github.com/vitaliiPsl/crappy-ai/internal/oauth/store"
)

type FileStore struct {
	file *oauthstore.File[oauth.Session]
}

func NewFileStore(path string) (*FileStore, error) {
	file, err := oauthstore.NewFile[oauth.Session](path, "sessions")
	if err != nil {
		return nil, err
	}

	return &FileStore{file: file}, nil
}

func (s *FileStore) Load(ctx context.Context, key oauth.Key) (*oauth.Session, error) {
	return s.file.Load(ctx, key.ID())
}

func (s *FileStore) Save(ctx context.Context, key oauth.Key, session oauth.Session) error {
	return s.file.Save(ctx, key.ID(), session)
}

func (s *FileStore) Delete(ctx context.Context, key oauth.Key) error {
	return s.file.Delete(ctx, key.ID())
}
