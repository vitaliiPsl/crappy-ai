package settings

import (
	"fmt"
	"os"

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

	return NewStore(settings, expandedPath), nil
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
	}
}
