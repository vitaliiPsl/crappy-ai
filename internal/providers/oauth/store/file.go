package store

import (
	"context"

	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
	"github.com/vitaliiPsl/crappy-ai/internal/store/jsonfile"
)

type FileStore struct {
	file *jsonfile.File[provideroauth.Credential]
}

func NewFileStore(path string) (*FileStore, error) {
	file, err := jsonfile.New[provideroauth.Credential](path)
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
