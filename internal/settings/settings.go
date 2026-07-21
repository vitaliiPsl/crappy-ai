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
	DefaultMemoryPath   = "~/.crappy-ai/memory.json"
	DefaultOAuthPath    = "~/.crappy-ai/oauth.json"
	DefaultMCPOAuthPath = "~/.crappy-ai/mcp-oauth.json"
)

const (
	EnvSettingsPath = "CRAPPY_SETTINGS"
	EnvSessionsDir  = "CRAPPY_SESSIONS_DIR"
	EnvModelsPath   = "CRAPPY_MODELS_PATH"
	EnvSkillsPath   = "CRAPPY_SKILLS_PATH"
	EnvMemoryPath   = "CRAPPY_MEMORY_PATH"
)

type Settings struct {
	ConfigPath   string `yaml:"config_path,omitempty"`
	SessionsDir  string `yaml:"sessions_dir,omitempty"`
	ModelsPath   string `yaml:"models_path,omitempty"`
	SkillsPath   string `yaml:"skills_path,omitempty"`
	MemoryPath   string `yaml:"memory_path,omitempty"`
	OAuthPath    string `yaml:"oauth_path,omitempty"`
	MCPOAuthPath string `yaml:"mcp_oauth_path,omitempty"`

	Providers    []ProviderSettings           `yaml:"providers,omitempty"`
	ModelConfigs map[string][]kit.ModelConfig `yaml:"models,omitempty"`
	MCPClients   []mcp.Config                 `yaml:"mcp,omitempty"`

	Models map[string][]kit.ModelConfig `yaml:"-"`
}

func defaults() Settings {
	return Settings{
		ConfigPath:   DefaultConfigPath,
		SessionsDir:  DefaultSessionsDir,
		ModelsPath:   DefaultModelsPath,
		SkillsPath:   DefaultSkillsPath,
		MemoryPath:   DefaultMemoryPath,
		OAuthPath:    DefaultOAuthPath,
		MCPOAuthPath: DefaultMCPOAuthPath,
		Providers:    DefaultProviders(),
	}
}
