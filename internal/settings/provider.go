package settings

import (
	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
	"github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

const (
	ProviderAuthAPIKey = "api_key"
	ProviderAuthOAuth  = "oauth"
)

type ProviderAuthType string

type ProviderSettings struct {
	ID      string               `yaml:"id"`
	API     string               `yaml:"api"`
	BaseURL string               `yaml:"base_url,omitempty"`
	Auth    ProviderAuthSettings `yaml:"auth"`
}

type ProviderAuthSettings struct {
	Type ProviderAuthType `yaml:"type"`

	// API Key authentication settings
	APIKey    string `yaml:"api_key,omitempty"`
	APIKeyEnv string `yaml:"api_key_env,omitempty"`

	// OAuth authentication settings
	Driver string               `yaml:"driver,omitempty"`
	OAuth  provideroauth.Config `yaml:",inline"`
}

func DefaultProviders() []ProviderSettings {
	return []ProviderSettings{
		{
			ID:  models.ProviderAnthropic,
			API: models.ProviderAnthropic,
			Auth: ProviderAuthSettings{
				Type:      ProviderAuthAPIKey,
				APIKeyEnv: "ANTHROPIC_API_KEY",
			},
		},
		{
			ID:  models.ProviderOpenAI,
			API: models.ProviderOpenAI,
			Auth: ProviderAuthSettings{
				Type:      ProviderAuthAPIKey,
				APIKeyEnv: "OPENAI_API_KEY",
			},
		},
		{
			ID:  models.ProviderGoogle,
			API: models.ProviderGoogle,
			Auth: ProviderAuthSettings{
				Type:      ProviderAuthAPIKey,
				APIKeyEnv: "GOOGLE_API_KEY",
			},
		},
	}
}
