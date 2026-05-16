package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type modelsFile struct {
	Models map[string][]kit.ModelConfig `json:"models"`
}

func Load(path string) map[string][]kit.ModelConfig {
	out := DefaultModels()
	if path == "" {
		return out
	}

	remote, err := read(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: read models file: %v\n", err)

		return out
	}

	for provider, fresh := range remote {
		if len(fresh) > 0 {
			out[provider] = fresh
		}
	}

	return out
}

func read(path string) (map[string][]kit.ModelConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}

	var file modelsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("decode models cache: %w", err)
	}

	out := make(map[string][]kit.ModelConfig, len(file.Models))
	for provider, entries := range file.Models {
		models := make([]kit.ModelConfig, 0, len(entries))
		for _, m := range entries {
			models = append(models, normalizeModelConfig(provider, m))
		}

		out[provider] = models
	}

	return out, nil
}

func write(path string, providers map[string][]kit.ModelConfig) error {
	file := modelsFile{
		Models: make(map[string][]kit.ModelConfig, len(providers)),
	}

	for provider, models := range providers {
		entries := make([]kit.ModelConfig, 0, len(models))
		for _, m := range models {
			entries = append(entries, normalizeModelConfig(provider, m))
		}

		file.Models[provider] = entries
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create models cache dir: %w", err)
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode models cache: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write models cache: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("replace models cache: %w", err)
	}

	return nil
}

func normalizeModelConfig(fallbackProvider string, cfg kit.ModelConfig) kit.ModelConfig {
	if cfg.Provider == "" {
		cfg.Provider = fallbackProvider
	}

	if cfg.InputLimit == 0 {
		cfg.InputLimit = cfg.ContextWindow
	}

	return cfg
}
