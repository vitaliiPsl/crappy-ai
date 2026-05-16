package settings

import (
	"context"
	"fmt"
	"os"

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
