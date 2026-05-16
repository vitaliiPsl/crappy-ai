package settings

import (
	"context"
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

	base := defaults()
	models.ApplyModels(utils.ExpandHome(base.ModelsPath), base.Providers)

	settings := merge(base, fileSettings)
	settings = merge(settings, fromEnv())

	settings.ConfigPath = utils.ExpandHome(settings.ConfigPath)
	settings.SessionsDir = utils.ExpandHome(settings.SessionsDir)
	settings.ModelsPath = utils.ExpandHome(settings.ModelsPath)

	return NewStore(settings, expandedPath), nil
}

func RefreshModels(ctx context.Context, s Settings) error {
	return models.Refresh(ctx, s.ModelsPath, s.Providers)
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
		base.Providers = mergeProviders(base.Providers, overlay.Providers)
	}

	return base
}

func mergeProviders(base, overlay []models.ProviderSettings) []models.ProviderSettings {
	byName := make(map[string]models.ProviderSettings, len(base)+len(overlay))
	for _, p := range base {
		byName[p.Name] = p
	}

	for _, p := range overlay {
		if existing, ok := byName[p.Name]; ok {
			byName[p.Name] = mergeProvider(existing, p)

			continue
		}

		byName[p.Name] = p
	}

	merged := make([]models.ProviderSettings, 0, len(byName))
	for _, p := range byName {
		merged = append(merged, p)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Name < merged[j].Name
	})

	return merged
}

func mergeProvider(base, overlay models.ProviderSettings) models.ProviderSettings {
	if overlay.Name != "" {
		base.Name = overlay.Name
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

	if len(overlay.Models) > 0 {
		base.Models = overlay.Models
	}

	return base
}

func cloneSettings(settings Settings) Settings {
	settings.Providers = cloneProviders(settings.Providers)

	return settings
}

func cloneProviders(providers []models.ProviderSettings) []models.ProviderSettings {
	if providers == nil {
		return nil
	}

	out := make([]models.ProviderSettings, len(providers))
	for i, p := range providers {
		out[i] = cloneProvider(p)
	}

	return out
}

func cloneProvider(provider models.ProviderSettings) models.ProviderSettings {
	if provider.Models != nil {
		provider.Models = append(provider.Models[:0:0], provider.Models...)
	}

	return provider
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
	}
}
