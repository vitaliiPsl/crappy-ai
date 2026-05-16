package settings

import "github.com/vitaliiPsl/crappy-ai/internal/settings/models"

type ProviderSettings struct {
	Name      string `yaml:"name"`
	API       string `yaml:"api"`
	BaseURL   string `yaml:"base_url,omitempty"`
	APIKey    string `yaml:"api_key,omitempty"`
	APIKeyEnv string `yaml:"api_key_env,omitempty"`
}

func DefaultProviders() []ProviderSettings {
	return []ProviderSettings{
		{
			Name:      models.ProviderAnthropic,
			API:       models.ProviderAnthropic,
			APIKeyEnv: "ANTHROPIC_API_KEY",
		},
		{
			Name:      models.ProviderOpenAI,
			API:       models.ProviderOpenAI,
			APIKeyEnv: "OPENAI_API_KEY",
		},
		{
			Name:      models.ProviderGoogle,
			API:       models.ProviderGoogle,
			APIKeyEnv: "GOOGLE_API_KEY",
		},
	}
}
