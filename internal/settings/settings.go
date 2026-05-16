package settings

import "github.com/vitaliiPsl/crappy-ai/internal/settings/models"

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

func merge(base, overlay Settings) Settings {
	if overlay.ConfigPath != "" {
		base.ConfigPath = overlay.ConfigPath
	}

	if overlay.SessionsDir != "" {
		base.SessionsDir = overlay.SessionsDir
	}

	if overlay.ModelsPath != "" {
		base.ModelsPath = overlay.ModelsPath
	}

	if len(overlay.Providers) > 0 {
		byName := make(map[string]models.ProviderSettings, len(base.Providers)+len(overlay.Providers))
		for _, p := range base.Providers {
			byName[p.Name] = p
		}

		for _, p := range overlay.Providers {
			byName[p.Name] = p
		}

		base.Providers = make([]models.ProviderSettings, 0, len(byName))
		for _, p := range byName {
			base.Providers = append(base.Providers, p)
		}
	}

	return base
}
