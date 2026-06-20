package settings

import (
	"fmt"
	"os"
	"sort"

	"github.com/vitaliiPsl/crappy-ai/internal/settings/models"
	"github.com/vitaliiPsl/crappy-ai/internal/utils"
)

func Load() (*Store, error) {
	expandedPath := utils.ExpandHome(resolvePath())

	fileSettings, exists, err := loadFile(expandedPath)
	if err != nil {
		return nil, err
	}

	if !exists {
		if err := writeFile(expandedPath, defaults()); err != nil {
			fmt.Fprintf(os.Stderr, "warning: init settings file: %v\n", err)
		}
	}

	settings := merge(defaults(), fileSettings)
	settings = merge(settings, fromEnv())

	settings.ConfigPath = utils.ExpandHome(settings.ConfigPath)
	settings.SessionsDir = utils.ExpandHome(settings.SessionsDir)
	settings.ModelsPath = utils.ExpandHome(settings.ModelsPath)
	settings.SkillsPath = utils.ExpandHome(settings.SkillsPath)
	settings.OAuthPath = utils.ExpandHome(settings.OAuthPath)

	settings.Models = models.Merge(models.Load(settings.ModelsPath), settings.ModelConfigs)

	return NewStore(settings, expandedPath), nil
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

	if overlay.SkillsPath != "" {
		base.SkillsPath = overlay.SkillsPath
	}

	if overlay.OAuthPath != "" {
		base.OAuthPath = overlay.OAuthPath
	}

	if len(overlay.Providers) > 0 {
		base.Providers = mergeProviders(base.Providers, overlay.Providers)
	}

	if len(overlay.ModelConfigs) > 0 {
		base.ModelConfigs = models.Merge(base.ModelConfigs, overlay.ModelConfigs)
	}

	if len(overlay.MCPClients) > 0 {
		base.MCPClients = overlay.MCPClients
	}

	return base
}

func mergeProviders(base, overlay []ProviderSettings) []ProviderSettings {
	byID := make(map[string]ProviderSettings, len(base)+len(overlay))
	for _, p := range base {
		byID[p.ID] = p
	}

	for _, p := range overlay {
		if existing, ok := byID[p.ID]; ok {
			byID[p.ID] = mergeProvider(existing, p)

			continue
		}

		byID[p.ID] = p
	}

	merged := make([]ProviderSettings, 0, len(byID))
	for _, p := range byID {
		merged = append(merged, p)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].ID < merged[j].ID
	})

	return merged
}

func mergeProvider(base, overlay ProviderSettings) ProviderSettings {
	if overlay.ID != "" {
		base.ID = overlay.ID
	}

	if overlay.API != "" {
		base.API = overlay.API
	}

	if overlay.BaseURL != "" {
		base.BaseURL = overlay.BaseURL
	}

	if overlay.APIKey != "" {
		base.APIKey = overlay.APIKey
	}

	if overlay.APIKeyEnv != "" {
		base.APIKeyEnv = overlay.APIKeyEnv
	}

	return base
}

func resolvePath() string {
	if path := os.Getenv(EnvSettingsPath); path != "" {
		return path
	}

	return DefaultSettingsPath
}

func fromEnv() Settings {
	return Settings{
		SessionsDir: os.Getenv(EnvSessionsDir),
		ModelsPath:  os.Getenv(EnvModelsPath),
		SkillsPath:  os.Getenv(EnvSkillsPath),
	}
}
