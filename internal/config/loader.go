package config

import (
	"fmt"
	"os"

	"github.com/vitaliiPsl/crappy-ai/internal/utils"
)

func Load(path string, flags Flags) (*Store, error) {
	fileCfg, exists, err := loadFile(path)
	if err != nil {
		return nil, err
	}

	if !exists {
		if err := writeFile(path, defaults()); err != nil {
			fmt.Fprintf(os.Stderr, "warning: init config file: %v\n", err)
		}
	}

	cfg := merge(defaults(), fileCfg)

	envCfg, err := fromEnv()
	if err != nil {
		return nil, err
	}

	flagCfg, err := fromFlags(flags)
	if err != nil {
		return nil, err
	}

	cfg = merge(cfg, envCfg)
	cfg = merge(cfg, flagCfg)

	return NewStore(cfg, path), nil
}

func fromEnv() (Config, error) {
	temperature, err := utils.ParseFloat32Ptr(os.Getenv(EnvTemperature))
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", EnvTemperature, err)
	}

	maxOutputTokens, err := utils.ParseNonnegativeInt32Ptr(os.Getenv(EnvMaxOutputTokens))
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", EnvMaxOutputTokens, err)
	}

	return Config{
		Agent: Agent{
			Provider:        os.Getenv(EnvProvider),
			Model:           os.Getenv(EnvModel),
			Thinking:        os.Getenv(EnvThinking),
			Temperature:     temperature,
			MaxOutputTokens: maxOutputTokens,
		},
		Mode: Mode(os.Getenv(EnvMode)),
	}, nil
}

func fromFlags(f Flags) (Config, error) {
	return Config{
		Agent: Agent{
			Provider: f.Provider,
			Model:    f.Model,
			Thinking: f.Thinking,
		},
		Mode: Mode(f.Mode),
		Cwd:  f.Cwd,
	}, nil
}
