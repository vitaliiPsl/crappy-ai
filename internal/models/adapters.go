package models

import (
	"github.com/vitaliiPsl/crappy-adk/kit"
	adkproviders "github.com/vitaliiPsl/crappy-adk/providers"
	"github.com/vitaliiPsl/crappy-adk/providers/anthropic"
	"github.com/vitaliiPsl/crappy-adk/providers/google"
	"github.com/vitaliiPsl/crappy-adk/providers/openai"

	settings "github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

type apiAdapter func(id string, opts ...adkproviders.ModelOption) (kit.Model, error)

var apiAdapters = map[string]apiAdapter{
	settings.ProviderAnthropic: func(id string, opts ...adkproviders.ModelOption) (kit.Model, error) {
		return anthropic.New(id, opts...)
	},
	settings.ProviderOpenAI: func(id string, opts ...adkproviders.ModelOption) (kit.Model, error) {
		return openai.New(id, opts...)
	},
	settings.ProviderGoogle: func(id string, opts ...adkproviders.ModelOption) (kit.Model, error) {
		return google.New(id, opts...)
	},
}
