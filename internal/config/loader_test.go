package config

import "testing"

func TestFromEnvReadsMode(t *testing.T) {
	t.Setenv(EnvMode, string(ModeYolo))

	cfg, err := fromEnv()
	if err != nil {
		t.Fatalf("fromEnv: %v", err)
	}

	got := cfg.Mode
	if got != ModeYolo {
		t.Fatalf("mode = %q, want %q", got, ModeYolo)
	}
}

func TestFromFlagsReadsMode(t *testing.T) {
	cfg, err := fromFlags(Flags{Mode: string(ModeYolo)})
	if err != nil {
		t.Fatalf("fromFlags: %v", err)
	}

	got := cfg.Mode
	if got != ModeYolo {
		t.Fatalf("mode = %q, want %q", got, ModeYolo)
	}
}

func TestFromEnvReadsGenerationConfig(t *testing.T) {
	t.Setenv(EnvTemperature, "0.25")
	t.Setenv(EnvMaxOutputTokens, "2048")

	cfg, err := fromEnv()
	if err != nil {
		t.Fatalf("fromEnv: %v", err)
	}

	if cfg.Temperature == nil || *cfg.Temperature != 0.25 {
		t.Fatalf("temperature = %v, want 0.25", cfg.Temperature)
	}

	if cfg.MaxOutputTokens == nil || *cfg.MaxOutputTokens != 2048 {
		t.Fatalf("max output tokens = %v, want 2048", cfg.MaxOutputTokens)
	}
}
