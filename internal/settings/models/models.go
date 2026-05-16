package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

const ModelsVersion = 1

type modelsFile struct {
	Version   int                           `json:"version"`
	Providers map[string]modelsFileProvider `json:"providers"`
}

type modelsFileProvider struct {
	Models []Model `json:"models"`
}

type Model struct {
	ID              string            `json:"id"`
	Provider        string            `json:"provider"`
	ContextWindow   int               `json:"context_window"`
	InputLimit      int               `json:"input_limit"`
	OutputLimit     int               `json:"output_limit"`
	Cost            ModelCost         `json:"cost"`
	Capabilities    ModelCapabilities `json:"capabilities"`
	KnowledgeCutoff string            `json:"knowledge_cutoff,omitempty"`
	ReleaseDate     string            `json:"release_date,omitempty"`
}

type ModelCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read,omitempty"`
	CacheWrite float64 `json:"cache_write,omitempty"`
}

type ModelCapabilities struct {
	Text      bool `json:"text,omitempty"`
	Image     bool `json:"image,omitempty"`
	Audio     bool `json:"audio,omitempty"`
	Video     bool `json:"video,omitempty"`
	PDF       bool `json:"pdf,omitempty"`
	Tools     bool `json:"tools,omitempty"`
	Streaming bool `json:"streaming,omitempty"`
	Reasoning bool `json:"reasoning,omitempty"`
	Caching   bool `json:"caching,omitempty"`
	Batch     bool `json:"batch,omitempty"`
}

func Read(path string) (map[string][]kit.ModelConfig, error) {
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

	if file.Version != ModelsVersion {
		return nil, fmt.Errorf("unsupported models cache version %d", file.Version)
	}

	out := make(map[string][]kit.ModelConfig, len(file.Providers))
	for provider, entry := range file.Providers {
		models := make([]kit.ModelConfig, 0, len(entry.Models))
		for _, m := range entry.Models {
			models = append(models, m.toKit(provider))
		}

		out[provider] = models
	}

	return out, nil
}

func writeModels(path string, providers map[string][]kit.ModelConfig) error {
	file := modelsFile{
		Version:   ModelsVersion,
		Providers: make(map[string]modelsFileProvider, len(providers)),
	}

	for provider, models := range providers {
		entries := make([]Model, 0, len(models))
		for _, m := range models {
			entries = append(entries, fromKit(m))
		}

		file.Providers[provider] = modelsFileProvider{Models: entries}
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

func ApplyModels(path string, providers []ProviderSettings) {
	if path == "" {
		return
	}

	remote, err := Read(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: read models file: %v\n", err)

		return
	}

	if len(remote) == 0 {
		return
	}

	for i, p := range providers {
		if fresh, ok := remote[p.Name]; ok && len(fresh) > 0 {
			providers[i].Models = fresh
		}
	}
}

func fromKit(m kit.ModelConfig) Model {
	return Model{
		ID:              m.ID,
		Provider:        m.Provider,
		ContextWindow:   m.ContextWindow,
		InputLimit:      m.InputLimit,
		OutputLimit:     m.OutputLimit,
		KnowledgeCutoff: m.KnowledgeCutoff,
		ReleaseDate:     m.ReleaseDate,
		Cost: ModelCost{
			Input:      m.Cost.Input,
			Output:     m.Cost.Output,
			CacheRead:  m.Cost.CacheRead,
			CacheWrite: m.Cost.CacheWrite,
		},
		Capabilities: ModelCapabilities{
			Text:      m.Capabilities.Text,
			Image:     m.Capabilities.Image,
			Audio:     m.Capabilities.Audio,
			Video:     m.Capabilities.Video,
			PDF:       m.Capabilities.PDF,
			Tools:     m.Capabilities.Tools,
			Streaming: m.Capabilities.Streaming,
			Reasoning: m.Capabilities.Reasoning,
			Caching:   m.Capabilities.Caching,
			Batch:     m.Capabilities.Batch,
		},
	}
}

func (e Model) toKit(fallbackProvider string) kit.ModelConfig {
	provider := e.Provider
	if provider == "" {
		provider = fallbackProvider
	}

	cfg := kit.ModelConfig{
		ID:              e.ID,
		Provider:        provider,
		ContextWindow:   e.ContextWindow,
		InputLimit:      e.InputLimit,
		OutputLimit:     e.OutputLimit,
		KnowledgeCutoff: e.KnowledgeCutoff,
		ReleaseDate:     e.ReleaseDate,
		Cost: kit.ModelCost{
			Input:      e.Cost.Input,
			Output:     e.Cost.Output,
			CacheRead:  e.Cost.CacheRead,
			CacheWrite: e.Cost.CacheWrite,
		},
		Capabilities: kit.ModelCapabilities{
			Text:      e.Capabilities.Text,
			Image:     e.Capabilities.Image,
			Audio:     e.Capabilities.Audio,
			Video:     e.Capabilities.Video,
			PDF:       e.Capabilities.PDF,
			Tools:     e.Capabilities.Tools,
			Streaming: e.Capabilities.Streaming,
			Reasoning: e.Capabilities.Reasoning,
			Caching:   e.Capabilities.Caching,
			Batch:     e.Capabilities.Batch,
		},
	}

	if cfg.InputLimit == 0 {
		cfg.InputLimit = cfg.ContextWindow
	}

	return cfg
}
