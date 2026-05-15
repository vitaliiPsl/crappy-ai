package config

import (
	"fmt"
	"os"
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
	cfg = merge(cfg, fromEnv())
	cfg = merge(cfg, fromFlags(flags))

	return NewStore(cfg, path), nil
}

func fromEnv() Config {
	return Config{
		Provider: os.Getenv(EnvProvider),
		Model:    os.Getenv(EnvModel),
		Thinking: os.Getenv(EnvThinking),
	}
}

func fromFlags(f Flags) Config {
	return Config{
		Provider: f.Provider,
		Model:    f.Model,
		Thinking: f.Thinking,
		Cwd:      f.Cwd,
	}
}
