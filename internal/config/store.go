package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type Store struct {
	mu   sync.RWMutex
	cfg  Config
	path string
}

func NewStore(cfg Config, path string) *Store {
	return &Store{
		cfg:  cfg,
		path: path,
	}
}

func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg
}

func (s *Store) Save(cfg Config) error {
	s.mu.Lock()
	s.cfg = cfg
	path := s.path
	s.mu.Unlock()

	return writeFile(path, cfg)
}

func (s *Store) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, _, err := loadFile(s.path)
	if err != nil {
		return fmt.Errorf("reload config: %w", err)
	}

	s.cfg = cfg

	return nil
}

func loadFile(path string) (Config, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, false, nil
		}

		return Config{}, false, fmt.Errorf("read config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, false, fmt.Errorf("parse config file %q: %w", path, err)
	}

	return cfg, true, nil
}

func writeFile(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	tmp := path + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)

	if err := enc.Encode(cfg); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)

		return fmt.Errorf("encode config: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("close config file: %w", err)
	}

	return os.Rename(tmp, path)
}
