package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Load(path string, flags Flags) (Config, error) {
	configPath := resolvePath(path)
	expandedPath := expandHome(configPath)

	fileCfg, exists, err := loadConfigFile(expandedPath)
	if err != nil {
		return Config{}, err
	}

	if !exists {
		if err := writeConfigFile(expandedPath, defaults()); err != nil {
			fmt.Fprintf(os.Stderr, "warning: init config file: %v\n", err)
		}
	}

	cfg := merge(defaults(), fileCfg)
	cfg = merge(cfg, fromEnv())
	cfg = merge(cfg, fromFlags(flags))

	cfg.ConfigPath = configPath
	cfg.SessionsDir = expandHome(cfg.SessionsDir)

	workDir, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("get working directory: %w", err)
	}

	cfg.WorkDir = workDir

	return cfg, nil
}

func resolvePath(path string) string {
	if path == "" {
		path = os.Getenv(EnvConfigPath)
	}

	if path == "" {
		path = DefaultConfigPath
	}

	return path
}

func fromEnv() Config {
	return Config{
		Provider:    os.Getenv(EnvProvider),
		Model:       os.Getenv(EnvModel),
		SessionsDir: os.Getenv(EnvSessionsDir),
		Thinking:    os.Getenv(EnvThinking),
	}
}

func fromFlags(f Flags) Config {
	return Config{
		Provider:    f.Provider,
		Model:       f.Model,
		SessionsDir: f.SessionsDir,
		Thinking:    f.Thinking,
	}
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, path[2:])
}
