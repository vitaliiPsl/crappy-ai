package config

import (
	_ "embed"

	permissionmodel "github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	settingsmodels "github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

//go:embed system_prompt.md
var DefaultSystemPrompt string

func defaults() Config {
	return Config{
		SystemPrompt: DefaultSystemPrompt,
		Mode:         ModeDefault,
		Provider:     settingsmodels.ProviderGoogle,
		Model:        "gemini-3.1-flash-lite",
		Thinking:     "medium",
		Permissions: permissionmodel.Permissions{
			Default: permissionmodel.Ask,
			Allow: []permissionmodel.Rule{
				{Tool: "list", Pattern: "./**"},
				{Tool: "read_file", Pattern: "./**"},
			},
		},
	}
}
