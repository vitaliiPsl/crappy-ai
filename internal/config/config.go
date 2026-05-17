package config

import "github.com/vitaliiPsl/crappy-ai/internal/permission"

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

	Permissions permission.Permissions `yaml:"permissions,omitempty"`

	Cwd string `yaml:"-"`
}

type Flags struct {
	Provider string
	Model    string
	Thinking string
	Cwd      string
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

	base.Permissions = permission.Merge(base.Permissions, overlay.Permissions)

	if overlay.Cwd != "" {
		base.Cwd = overlay.Cwd
	}

	return base
}
