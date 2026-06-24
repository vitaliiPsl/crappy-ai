package config

import "github.com/vitaliiPsl/crappy-ai/internal/permission/model"

const (
	EnvProvider        = "CRAPPY_PROVIDER"
	EnvModel           = "CRAPPY_MODEL"
	EnvThinking        = "CRAPPY_THINKING"
	EnvTemperature     = "CRAPPY_TEMPERATURE"
	EnvMaxOutputTokens = "CRAPPY_MAX_OUTPUT_TOKENS"
	EnvMode            = "CRAPPY_MODE"
)

const RootAgentName = "root"

type Mode string

const (
	ModeDefault Mode = "default"
	ModeYolo    Mode = "yolo"
)

type Agent struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`

	Prompt string `yaml:"prompt,omitempty"`

	Model           string   `yaml:"model,omitempty"`
	Provider        string   `yaml:"provider,omitempty"`
	Thinking        string   `yaml:"thinking,omitempty"`
	Temperature     *float32 `yaml:"temperature,omitempty"`
	MaxOutputTokens *int32   `yaml:"max_output_tokens,omitempty"`

	Tools       []string          `yaml:"tools,omitempty"`
	Permissions model.Permissions `yaml:"permissions,omitempty"`
}

type Config struct {
	Agent `yaml:",inline"`

	Mode Mode `yaml:"mode,omitempty"`

	Agents []Agent `yaml:"agents,omitempty"`

	Cwd string `yaml:"-"`
}

type Flags struct {
	Provider string
	Model    string
	Thinking string
	Mode     string
	Cwd      string
}

func (c Config) Subagent(name string) (Agent, bool) {
	for _, sub := range c.Agents {
		if sub.Name == name {
			return c.resolveSubagent(sub), true
		}
	}

	return Agent{}, false
}

func (c Config) resolveSubagent(sub Agent) Agent {
	if sub.Provider == "" {
		sub.Provider = c.Provider
	}

	if sub.Model == "" {
		sub.Model = c.Model
	}

	if sub.Thinking == "" {
		sub.Thinking = c.Thinking
	}

	if sub.Temperature == nil {
		sub.Temperature = c.Temperature
	}

	if sub.MaxOutputTokens == nil {
		sub.MaxOutputTokens = c.MaxOutputTokens
	}

	return sub
}

func merge(base, overlay Config) Config {
	base.Agent = mergeAgent(base.Agent, overlay.Agent)

	if overlay.Mode != "" {
		base.Mode = overlay.Mode
	}

	if len(overlay.Agents) > 0 {
		base.Agents = overlay.Agents
	}

	if overlay.Cwd != "" {
		base.Cwd = overlay.Cwd
	}

	return base
}

func mergeAgent(base, overlay Agent) Agent {
	if overlay.Name != "" {
		base.Name = overlay.Name
	}

	if overlay.Description != "" {
		base.Description = overlay.Description
	}

	if overlay.Prompt != "" {
		base.Prompt = overlay.Prompt
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

	if overlay.Temperature != nil {
		base.Temperature = overlay.Temperature
	}

	if overlay.MaxOutputTokens != nil {
		base.MaxOutputTokens = overlay.MaxOutputTokens
	}

	if len(overlay.Tools) > 0 {
		base.Tools = overlay.Tools
	}

	base.Permissions = model.Merge(base.Permissions, overlay.Permissions)

	return base
}
