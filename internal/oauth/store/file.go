package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var fileLocks sync.Map

type File[T any] struct {
	mu    *sync.Mutex
	path  string
	field string
}

type fileData[T any] struct {
	envelope map[string]json.RawMessage
	entries  map[string]T
}

func NewFile[T any](path, field string) (*File[T], error) {
	if path == "" {
		return nil, errors.New("oauth path is required")
	}

	if field == "" {
		return nil, errors.New("oauth file field is required")
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve oauth path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create oauth dir: %w", err)
	}

	lock, _ := fileLocks.LoadOrStore(path, &sync.Mutex{})

	return &File[T]{mu: lock.(*sync.Mutex), path: path, field: field}, nil
}

func (f *File[T]) Load(_ context.Context, key string) (*T, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := f.read()
	if err != nil {
		return nil, err
	}

	entry, ok := file.entries[key]
	if !ok {
		return nil, nil
	}

	return &entry, nil
}

func (f *File[T]) Save(_ context.Context, key string, value T) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := f.read()
	if err != nil {
		return err
	}

	file.entries[key] = value

	return f.write(file)
}

func (f *File[T]) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := f.read()
	if err != nil {
		return err
	}

	if _, ok := file.entries[key]; !ok {
		return nil
	}

	delete(file.entries, key)

	return f.write(file)
}

func (f *File[T]) read() (fileData[T], error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileData[T]{
				envelope: make(map[string]json.RawMessage),
				entries:  make(map[string]T),
			}, nil
		}

		return fileData[T]{}, fmt.Errorf("read oauth file: %w", err)
	}

	if len(data) == 0 {
		return fileData[T]{
			envelope: make(map[string]json.RawMessage),
			entries:  make(map[string]T),
		}, nil
	}

	envelope := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fileData[T]{}, fmt.Errorf("parse oauth file: %w", err)
	}

	entries := make(map[string]T)
	if raw := envelope[f.field]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &entries); err != nil {
			return fileData[T]{}, fmt.Errorf("parse oauth file field %q: %w", f.field, err)
		}
	}

	return fileData[T]{envelope: envelope, entries: entries}, nil
}

func (f *File[T]) write(file fileData[T]) error {
	entries, err := json.Marshal(file.entries)
	if err != nil {
		return fmt.Errorf("encode oauth file field %q: %w", f.field, err)
	}

	file.envelope[f.field] = entries

	data, err := json.MarshalIndent(file.envelope, "", "  ")
	if err != nil {
		return fmt.Errorf("encode oauth file: %w", err)
	}

	data = append(data, '\n')

	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write oauth file: %w", err)
	}

	if err := os.Rename(tmp, f.path); err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("replace oauth file: %w", err)
	}

	return nil
}
