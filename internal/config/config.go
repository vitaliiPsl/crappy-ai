package config

const (
	EnvProvider = "CRAPPY_PROVIDER"
	EnvModel    = "CRAPPY_MODEL"
	EnvThinking = "CRAPPY_THINKING"
)

type Config struct {
	SystemPrompt string `yaml:"system_prompt,omitempty"`
	Provider     string `yaml:"provider"`
	Model        string `yaml:"model"`
	Thinking     string `yaml:"thinking,omitempty"`
}

type Flags struct {
	Provider string
	Model    string
	Thinking string
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

	return base
}
