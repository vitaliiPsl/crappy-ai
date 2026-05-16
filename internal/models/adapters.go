package models

import (
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/providers/anthropic"
	"github.com/vitaliiPsl/crappy-adk/providers/google"
	"github.com/vitaliiPsl/crappy-adk/providers/openai"

	settings "github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

type apiAdapter func(apiKey, baseURL, modelID string, config kit.ModelConfig) (kit.Model, error)

var apiAdapters = map[string]apiAdapter{
	settings.ProviderAnthropic: func(apiKey, baseURL, modelID string, config kit.ModelConfig) (kit.Model, error) {
		opts := []anthropic.Option{anthropic.WithModelConfig(config)}
		if baseURL != "" {
			opts = append(opts, anthropic.WithBaseURL(baseURL))
		}

		return anthropic.New(apiKey, modelID, opts...)
	},
	settings.ProviderOpenAI: func(apiKey, baseURL, modelID string, config kit.ModelConfig) (kit.Model, error) {
		opts := []openai.Option{openai.WithModelConfig(config)}
		if baseURL != "" {
			opts = append(opts, openai.WithBaseURL(baseURL))
		}

		return openai.New(apiKey, modelID, opts...)
	},
	settings.ProviderGoogle: func(apiKey, baseURL, modelID string, config kit.ModelConfig) (kit.Model, error) {
		opts := []google.Option{google.WithModelConfig(config)}
		if baseURL != "" {
			opts = append(opts, google.WithBaseURL(baseURL))
		}

		return google.New(apiKey, modelID, opts...)
	},
}
