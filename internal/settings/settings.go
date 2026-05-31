package settings

import (
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
)

const (
	DefaultSettingsPath = "~/.crappy-ai/settings.yaml"
	DefaultConfigPath   = "~/.crappy-ai/config.yaml"
	DefaultSessionsDir  = "~/.crappy-ai/sessions"
	DefaultModelsPath   = "~/.crappy-ai/models.json"
	DefaultSkillsPath   = "~/.crappy-ai/skills"
)

const (
	EnvSettingsPath = "CRAPPY_SETTINGS"
	EnvSessionsDir  = "CRAPPY_SESSIONS_DIR"
	EnvModelsPath   = "CRAPPY_MODELS_PATH"
	EnvSkillsPath   = "CRAPPY_SKILLS_PATH"
)

type Settings struct {
	ConfigPath  string `yaml:"config_path,omitempty"`
	SessionsDir string `yaml:"sessions_dir,omitempty"`
	ModelsPath  string `yaml:"models_path,omitempty"`
	SkillsPath  string `yaml:"skills_path,omitempty"`

	Providers    []ProviderSettings           `yaml:"providers,omitempty"`
	ModelConfigs map[string][]kit.ModelConfig `yaml:"models,omitempty"`
	MCPClients   []mcp.Config                 `yaml:"mcp,omitempty"`

	Models map[string][]kit.ModelConfig `yaml:"-"`
}

func defaults() Settings {
	return Settings{
		ConfigPath:  DefaultConfigPath,
		SessionsDir: DefaultSessionsDir,
		ModelsPath:  DefaultModelsPath,
		SkillsPath:  DefaultSkillsPath,
		Providers:   DefaultProviders(),
	}
}
