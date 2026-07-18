package store

import (
	"context"

	appoauthstore "github.com/vitaliiPsl/crappy-ai/internal/oauth/store"
	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

type FileStore struct {
	file *appoauthstore.File[provideroauth.Credential]
}

func NewFileStore(path string) (*FileStore, error) {
	file, err := appoauthstore.NewFile[provideroauth.Credential](path, "credentials")
	if err != nil {
		return nil, err
	}

	return &FileStore{file: file}, nil
}

func (s *FileStore) Load(ctx context.Context, providerID string) (*provideroauth.Credential, error) {
	return s.file.Load(ctx, providerID)
}

func (s *FileStore) Save(ctx context.Context, providerID string, credential provideroauth.Credential) error {
	return s.file.Save(ctx, providerID, credential)
}

func (s *FileStore) Delete(ctx context.Context, providerID string) error {
	return s.file.Delete(ctx, providerID)
}
