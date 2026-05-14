package settings

import "github.com/vitaliiPsl/crappy-adk/kit"

const (
	DefaultSettingsPath = "~/.crappy-ai/settings.yaml"
	DefaultConfigPath   = "~/.crappy-ai/config.yaml"
	DefaultSessionsDir  = "~/.crappy-ai/sessions"
)

const (
	EnvSettingsPath = "CRAPPY_SETTINGS"
	EnvSessionsDir  = "CRAPPY_SESSIONS_DIR"
)

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

type Settings struct {
	ConfigPath  string             `yaml:"config_path,omitempty"`
	SessionsDir string             `yaml:"sessions_dir,omitempty"`
	Providers   []ProviderSettings `yaml:"providers,omitempty"`
}

func merge(base, overlay Settings) Settings {
	if overlay.ConfigPath != "" {
		base.ConfigPath = overlay.ConfigPath
	}

	if overlay.SessionsDir != "" {
		base.SessionsDir = overlay.SessionsDir
	}

	if len(overlay.Providers) > 0 {
		byName := make(map[string]ProviderSettings, len(base.Providers)+len(overlay.Providers))
		for _, p := range base.Providers {
			byName[p.Name] = p
		}

		for _, p := range overlay.Providers {
			byName[p.Name] = p
		}

		base.Providers = make([]ProviderSettings, 0, len(byName))
		for _, p := range byName {
			base.Providers = append(base.Providers, p)
		}
	}

	return base
}
