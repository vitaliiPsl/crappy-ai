package settings

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

type Store struct {
	mu       sync.RWMutex
	settings Settings
	path     string
}

func NewStore(settings Settings, path string) *Store {
	return &Store{
		settings: settings,
		path:     path,
	}
}

func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.settings
}

func (s *Store) Save(settings Settings) error {
	s.mu.Lock()
	s.settings = settings
	path := s.path
	s.mu.Unlock()

	return writeFile(path, settings)
}

func (s *Store) RefreshModels(ctx context.Context) error {
	fresh, err := models.Refresh(ctx, s.Get().ModelsPath)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.settings.Models = models.Merge(fresh, s.settings.ModelConfigs)
	s.mu.Unlock()

	return nil
}

func loadFile(path string) (Settings, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Settings{}, false, nil
		}

		return Settings{}, false, fmt.Errorf("read settings file %q: %w", path, err)
	}

	var settings Settings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return Settings{}, false, fmt.Errorf("parse settings file %q: %w", path, err)
	}

	return settings, true, nil
}

func writeFile(path string, settings Settings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}

	tmp := path + ".tmp"

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create settings file: %w", err)
	}

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)

	if err := enc.Encode(settings); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)

		return fmt.Errorf("encode settings: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("close settings file: %w", err)
	}

	return os.Rename(tmp, path)
}
