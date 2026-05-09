package models

import (
	"fmt"
	"os"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/providers/anthropic"
	"github.com/vitaliiPsl/crappy-adk/providers/google"
	"github.com/vitaliiPsl/crappy-adk/providers/openai"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
)

func BuildModel(s settings.Settings, cfg config.Config) (kit.Model, error) {
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

	apiKey := provider.APIKey
	if apiKey == "" && provider.APIKeyEnv != "" {
		apiKey = os.Getenv(provider.APIKeyEnv)
	}

	if apiKey == "" {
		return nil, fmt.Errorf("provider %q: no API key (set %s)", provider.Name, provider.APIKeyEnv)
	}

	switch provider.API {
	case settings.ProviderAnthropic:
		var opts []anthropic.Option
		if provider.BaseURL != "" {
			opts = append(opts, anthropic.WithBaseURL(provider.BaseURL))
		}

		return anthropic.New(apiKey, cfg.Model, opts...)
	case settings.ProviderOpenAI:
		var opts []openai.Option
		if provider.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(provider.BaseURL))
		}

		return openai.New(apiKey, cfg.Model, opts...)
	case settings.ProviderGoogle:
		var opts []google.Option
		if provider.BaseURL != "" {
			opts = append(opts, google.WithBaseURL(provider.BaseURL))
		}

		return google.New(apiKey, cfg.Model, opts...)
	default:
		return nil, fmt.Errorf("provider %q: unknown api %q", provider.Name, provider.API)
	}
}

func findProvider(providers []settings.ProviderSettings, name string) (settings.ProviderSettings, bool) {
	for _, p := range providers {
		if p.Name == name {
			return p, true
		}
	}

	return settings.ProviderSettings{}, false
}
