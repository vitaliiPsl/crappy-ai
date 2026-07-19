package jsonfile

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
	mu   *sync.Mutex
	path string
}

func New[T any](path string) (*File[T], error) {
	if path == "" {
		return nil, errors.New("file path is required")
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve file path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create file directory: %w", err)
	}

	lock, _ := fileLocks.LoadOrStore(path, &sync.Mutex{})

	return &File[T]{mu: lock.(*sync.Mutex), path: path}, nil
}

func (f *File[T]) Load(_ context.Context, key string) (*T, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := f.read()
	if err != nil {
		return nil, err
	}

	entry, ok := entries[key]
	if !ok {
		return nil, nil
	}

	return &entry, nil
}

func (f *File[T]) Save(_ context.Context, key string, value T) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := f.read()
	if err != nil {
		return err
	}

	entries[key] = value

	return f.write(entries)
}

func (f *File[T]) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := f.read()
	if err != nil {
		return err
	}

	if _, ok := entries[key]; !ok {
		return nil
	}

	delete(entries, key)

	return f.write(entries)
}

func (f *File[T]) read() (map[string]T, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]T), nil
		}

		return nil, fmt.Errorf("read JSON file: %w", err)
	}

	if len(data) == 0 {
		return make(map[string]T), nil
	}

	entries := make(map[string]T)
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse JSON file: %w", err)
	}

	return entries, nil
}

func (f *File[T]) write(entries map[string]T) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JSON file: %w", err)
	}

	data = append(data, '\n')

	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write JSON file: %w", err)
	}

	if err := os.Rename(tmp, f.path); err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("replace JSON file: %w", err)
	}

	return nil
}
