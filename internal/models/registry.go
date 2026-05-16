package models

import (
	"fmt"
	"os"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
	settingsmodels "github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

type Registry struct {
	settingsStore *settings.Store
}

func NewRegistry(settingsStore *settings.Store) *Registry {
	return &Registry{settingsStore: settingsStore}
}

func (r *Registry) GetProviders() []settingsmodels.ProviderSettings {
	return r.settingsStore.Get().Providers
}

func (r *Registry) GetProvider(name string) (settingsmodels.ProviderSettings, error) {
	p, ok := findProvider(r.settingsStore.Get().Providers, name)
	if !ok {
		return settingsmodels.ProviderSettings{}, fmt.Errorf("unknown provider %q", name)
	}

	return p, nil
}

func (r *Registry) Build(cfg config.Config) (kit.Model, error) {
	return buildModel(r.settingsStore.Get(), cfg)
}

func buildModel(s settings.Settings, cfg config.Config) (kit.Model, error) {
	if cfg.Provider == "" {
		return nil, fmt.Errorf("config: provider is not set")
	}

	if cfg.Model == "" {
		return nil, fmt.Errorf("config: model is not set")
	}

	provider, ok := findProvider(s.Providers, cfg.Provider)
	if !ok {
		return nil, fmt.Errorf("settings: unknown provider %q", cfg.Provider)
	}

	adapter, ok := apiAdapters[provider.API]
	if !ok {
		return nil, fmt.Errorf("provider %q: unknown api %q", provider.Name, provider.API)
	}

	apiKey := provider.APIKey
	if apiKey == "" && provider.APIKeyEnv != "" {
		apiKey = os.Getenv(provider.APIKeyEnv)
	}

	if apiKey == "" {
		return nil, fmt.Errorf("provider %q: no API key (set %s)", provider.Name, provider.APIKeyEnv)
	}

	modelConfig := findModelConfig(provider, cfg.Model)

	return adapter(apiKey, provider.BaseURL, cfg.Model, modelConfig)
}

func findProvider(providers []settingsmodels.ProviderSettings, name string) (settingsmodels.ProviderSettings, bool) {
	for _, p := range providers {
		if p.Name == name {
			return p, true
		}
	}

	return settingsmodels.ProviderSettings{}, false
}

func findModelConfig(provider settingsmodels.ProviderSettings, modelID string) kit.ModelConfig {
	for _, model := range provider.Models {
		if model.ID == modelID {
			return model
		}
	}

	return kit.ModelConfig{ID: modelID}
}
