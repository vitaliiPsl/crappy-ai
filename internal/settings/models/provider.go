package models

import "github.com/vitaliiPsl/crappy-adk/kit"

const (
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
	ProviderGoogle    = "google"
)

type ProviderSettings struct {
	Name      string            `yaml:"name"`
	API       string            `yaml:"api"`
	BaseURL   string            `yaml:"base_url,omitempty"`
	APIKey    string            `yaml:"api_key,omitempty"`
	APIKeyEnv string            `yaml:"api_key_env,omitempty"`
	Models    []kit.ModelConfig `yaml:"models,omitempty"`
}
