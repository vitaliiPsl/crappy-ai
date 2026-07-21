package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/vitaliiPsl/crappy-ai/internal/memory"
)

const version = 1

type document struct {
	Version  int             `json:"version"`
	Memories []memory.Memory `json:"memories"`
}

type FileStore struct {
	mu   sync.Mutex
	path string
	now  func() time.Time
}

func NewFileStore(path string) (*FileStore, error) {
	if path == "" {
		return nil, errors.New("memory path is required")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve memory path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		return nil, fmt.Errorf("create memory directory: %w", err)
	}

	return &FileStore{path: abs, now: time.Now}, nil
}

func (s *FileStore) List(_ context.Context) ([]memory.Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.read()
	if err != nil {
		return nil, err
	}

	return append([]memory.Memory(nil), doc.Memories...), nil
}

func (s *FileStore) Create(_ context.Context, params memory.CreateParams) (memory.Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	content, err := validate(params.Kind, params.Content)
	if err != nil {
		return memory.Memory{}, err
	}

	doc, err := s.read()
	if err != nil {
		return memory.Memory{}, err
	}

	for _, existing := range doc.Memories {
		if existing.Kind == params.Kind && normalize(existing.Content) == normalize(content) {
			return memory.Memory{}, errors.New("memory already exists")
		}
	}

	now := s.now().UTC()
	created := memory.Memory{
		ID: uuid.NewString(), Kind: params.Kind, Content: content, CreatedAt: now, UpdatedAt: now,
	}
	doc.Memories = append(doc.Memories, created)

	if err := s.write(doc); err != nil {
		return memory.Memory{}, err
	}

	return created, nil
}

func (s *FileStore) Update(_ context.Context, params memory.UpdateParams) (memory.Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	content, err := validate(params.Kind, params.Content)
	if err != nil {
		return memory.Memory{}, err
	}

	doc, err := s.read()
	if err != nil {
		return memory.Memory{}, err
	}

	for _, existing := range doc.Memories {
		if existing.ID != params.ID && existing.Kind == params.Kind && normalize(existing.Content) == normalize(content) {
			return memory.Memory{}, errors.New("memory already exists")
		}
	}

	for i := range doc.Memories {
		if doc.Memories[i].ID != params.ID {
			continue
		}

		doc.Memories[i].Kind = params.Kind
		doc.Memories[i].Content = content
		doc.Memories[i].UpdatedAt = s.now().UTC()
		updated := doc.Memories[i]

		if err := s.write(doc); err != nil {
			return memory.Memory{}, err
		}

		return updated, nil
	}

	return memory.Memory{}, fmt.Errorf("memory %q not found", params.ID)
}

func (s *FileStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.read()
	if err != nil {
		return err
	}

	for i := range doc.Memories {
		if doc.Memories[i].ID != id {
			continue
		}

		doc.Memories = append(doc.Memories[:i], doc.Memories[i+1:]...)

		return s.write(doc)
	}

	return fmt.Errorf("memory %q not found", id)
}

func (s *FileStore) read() (document, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return document{Version: version}, nil
	}

	if err != nil {
		return document{}, fmt.Errorf("read memory file: %w", err)
	}

	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return document{}, fmt.Errorf("parse memory file: %w", err)
	}

	if doc.Version != version {
		return document{}, fmt.Errorf("unsupported memory file version %d", doc.Version)
	}

	return doc, nil
}

func (s *FileStore) write(doc document) error {
	doc.Version = version

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode memory file: %w", err)
	}

	data = append(data, '\n')

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write memory file: %w", err)
	}

	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("replace memory file: %w", err)
	}

	return nil
}

func validate(kind memory.Kind, content string) (string, error) {
	if !kind.Valid() {
		return "", fmt.Errorf("invalid memory kind %q", kind)
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return "", errors.New("memory content is required")
	}

	return content, nil
}

func normalize(content string) string {
	return strings.ToLower(strings.Join(strings.Fields(content), " "))
}
