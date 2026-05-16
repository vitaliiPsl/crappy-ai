package settings

import (
	"github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

const (
	DefaultSettingsPath = "~/.crappy-ai/settings.yaml"
	DefaultConfigPath   = "~/.crappy-ai/config.yaml"
	DefaultSessionsDir  = "~/.crappy-ai/sessions"
	DefaultModelsPath   = "~/.crappy-ai/models.json"
)

const (
	EnvSettingsPath = "CRAPPY_SETTINGS"
	EnvSessionsDir  = "CRAPPY_SESSIONS_DIR"
	EnvModelsPath   = "CRAPPY_MODELS_PATH"
)

type Settings struct {
	ConfigPath  string                    `yaml:"config_path,omitempty"`
	SessionsDir string                    `yaml:"sessions_dir,omitempty"`
	ModelsPath  string                    `yaml:"models_path,omitempty"`
	Providers   []models.ProviderSettings `yaml:"providers,omitempty"`
}

func defaults() Settings {
	return Settings{
		ConfigPath:  DefaultConfigPath,
		SessionsDir: DefaultSessionsDir,
		ModelsPath:  DefaultModelsPath,
		Providers:   models.DefaultProviders(),
	}
}
