package models

import (
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/providers/anthropic"
	"github.com/vitaliiPsl/crappy-adk/providers/google"
	"github.com/vitaliiPsl/crappy-adk/providers/openai"

	"github.com/vitaliiPsl/crappy-ai/internal/settings"
)

type apiAdapter func(apiKey, baseURL, modelID string) (kit.Model, error)

var apiAdapters = map[string]apiAdapter{
	settings.ProviderAnthropic: func(apiKey, baseURL, modelID string) (kit.Model, error) {
		var opts []anthropic.Option
		if baseURL != "" {
			opts = append(opts, anthropic.WithBaseURL(baseURL))
		}

		return anthropic.New(apiKey, modelID, opts...)
	},
	settings.ProviderOpenAI: func(apiKey, baseURL, modelID string) (kit.Model, error) {
		var opts []openai.Option
		if baseURL != "" {
			opts = append(opts, openai.WithBaseURL(baseURL))
		}

		return openai.New(apiKey, modelID, opts...)
	},
	settings.ProviderGoogle: func(apiKey, baseURL, modelID string) (kit.Model, error) {
		var opts []google.Option
		if baseURL != "" {
			opts = append(opts, google.WithBaseURL(baseURL))
		}

		return google.New(apiKey, modelID, opts...)
	},
}
