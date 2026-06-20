package models

import (
	"fmt"
	"os"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/settings"
)

type Registry struct {
	settingsStore *settings.Store
}

func NewRegistry(settingsStore *settings.Store) *Registry {
	return &Registry{
		settingsStore: settingsStore,
	}
}

func (r *Registry) GetProviders() []settings.ProviderSettings {
	return r.settingsStore.Get().Providers
}

func (r *Registry) GetProvider(id string) (settings.ProviderSettings, error) {
	p, ok := findProvider(r.settingsStore.Get().Providers, id)
	if !ok {
		return settings.ProviderSettings{}, fmt.Errorf("unknown provider %q", id)
	}

	return p, nil
}

func (r *Registry) Build(provider, model string) (kit.Model, error) {
	return buildModel(r.settingsStore.Get(), provider, model)
}

func buildModel(s settings.Settings, providerName, modelID string) (kit.Model, error) {
	if providerName == "" {
		return nil, fmt.Errorf("config: provider is not set")
	}

	if modelID == "" {
		return nil, fmt.Errorf("config: model is not set")
	}

	provider, ok := findProvider(s.Providers, providerName)
	if !ok {
		return nil, fmt.Errorf("settings: unknown provider %q", providerName)
	}

	adapter, ok := apiAdapters[provider.API]
	if !ok {
		return nil, fmt.Errorf("provider %q: unknown api %q", provider.ID, provider.API)
	}

	apiKey := provider.APIKey
	if apiKey == "" && provider.APIKeyEnv != "" {
		apiKey = os.Getenv(provider.APIKeyEnv)
	}

	if apiKey == "" {
		return nil, fmt.Errorf("provider %q: no API key (set %s)", provider.ID, provider.APIKeyEnv)
	}

	modelConfig := findModel(s.Models[providerName], modelID)

	return adapter(apiKey, provider.BaseURL, modelID, modelConfig)
}

func findProvider(providers []settings.ProviderSettings, id string) (settings.ProviderSettings, bool) {
	for _, p := range providers {
		if p.ID == id {
			return p, true
		}
	}

	return settings.ProviderSettings{}, false
}

func findModel(models []kit.ModelConfig, modelID string) kit.ModelConfig {
	for _, model := range models {
		if model.ID == modelID {
			return model
		}
	}

	return kit.ModelConfig{ID: modelID}
}
