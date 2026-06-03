package tokenstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth"
)

type FileStore struct {
	mu   sync.Mutex
	path string
}

func NewFileStore(path string) (*FileStore, error) {
	if path == "" {
		return nil, fmt.Errorf("oauth path is required")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create oauth dir: %w", err)
	}

	return &FileStore{path: path}, nil
}

func (s *FileStore) Load(_ context.Context, key oauth.SessionKey) (*oauth.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.read()
	if err != nil {
		return nil, err
	}

	session, ok := file.Sessions[key.ID()]
	if !ok {
		return nil, nil
	}

	return &session, nil
}

func (s *FileStore) Save(_ context.Context, key oauth.SessionKey, session oauth.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.read()
	if err != nil {
		return err
	}

	file.Sessions[key.ID()] = session

	return s.write(file)
}

func (s *FileStore) Delete(_ context.Context, key oauth.SessionKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.read()
	if err != nil {
		return err
	}

	if _, ok := file.Sessions[key.ID()]; !ok {
		return nil
	}

	delete(file.Sessions, key.ID())

	return s.write(file)
}

type fileData struct {
	Sessions map[string]oauth.Session `json:"sessions"`
}

func newFileData() fileData {
	return fileData{
		Sessions: make(map[string]oauth.Session),
	}
}

func (s *FileStore) read() (fileData, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newFileData(), nil
		}

		return fileData{}, fmt.Errorf("read oauth file: %w", err)
	}

	if len(data) == 0 {
		return newFileData(), nil
	}

	file := newFileData()
	if err := json.Unmarshal(data, &file); err != nil {
		return fileData{}, fmt.Errorf("parse oauth file: %w", err)
	}

	if file.Sessions == nil {
		file.Sessions = make(map[string]oauth.Session)
	}

	return file, nil
}

func (s *FileStore) write(file fileData) error {
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode oauth file: %w", err)
	}

	data = append(data, '\n')
	tmp := s.path + ".tmp"

	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write oauth file: %w", err)
	}

	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("replace oauth file: %w", err)
	}

	return nil
}
