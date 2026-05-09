package config

const (
	DefaultAssistantDir = "~/.crappy-ai"
	DefaultConfigPath   = DefaultAssistantDir + "/config.yaml"
	DefaultSessionsDir  = DefaultAssistantDir + "/sessions"
)

const (
	EnvConfigPath  = "CRAPPY_CONFIG"
	EnvProvider    = "CRAPPY_PROVIDER"
	EnvModel       = "CRAPPY_MODEL"
	EnvSessionsDir = "CRAPPY_SESSIONS_DIR"
	EnvThinking    = "CRAPPY_THINKING"
)

const (
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
	ProviderGoogle    = "google"
)

type ProviderConfig struct {
	Name      string `yaml:"name"`
	API       string `yaml:"api"`
	BaseURL   string `yaml:"base_url,omitempty"`
	APIKey    string `yaml:"api_key,omitempty"`
	APIKeyEnv string `yaml:"api_key_env,omitempty"`
}

type Config struct {
	SystemPrompt string `yaml:"system_prompt,omitempty"`

	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Thinking string `yaml:"thinking,omitempty"`

	Providers []ProviderConfig `yaml:"providers,omitempty"`

	SessionsDir string `yaml:"sessions_dir,omitempty"`

	ConfigPath string `yaml:"-"`
	WorkDir    string `yaml:"-"`
}

type Flags struct {
	Provider    string
	Model       string
	SessionsDir string
	Thinking    string
}

func merge(base, overlay Config) Config {
	if overlay.SystemPrompt != "" {
		base.SystemPrompt = overlay.SystemPrompt
	}

	if overlay.Provider != "" {
		base.Provider = overlay.Provider
	}

	if overlay.Model != "" {
		base.Model = overlay.Model
	}

	if overlay.Thinking != "" {
		base.Thinking = overlay.Thinking
	}

	if overlay.SessionsDir != "" {
		base.SessionsDir = overlay.SessionsDir
	}

	if len(overlay.Providers) > 0 {
		byName := make(map[string]ProviderConfig, len(base.Providers)+len(overlay.Providers))
		for _, p := range base.Providers {
			byName[p.Name] = p
		}

		for _, p := range overlay.Providers {
			byName[p.Name] = p
		}

		base.Providers = make([]ProviderConfig, 0, len(byName))
		for _, p := range byName {
			base.Providers = append(base.Providers, p)
		}
	}

	return base
}
