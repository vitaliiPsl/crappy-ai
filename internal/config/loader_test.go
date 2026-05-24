package config

import "testing"

func TestFromEnvReadsMode(t *testing.T) {
	t.Setenv(EnvMode, string(ModeYolo))

	got := fromEnv().Mode
	if got != ModeYolo {
		t.Fatalf("mode = %q, want %q", got, ModeYolo)
	}
}

func TestFromFlagsReadsMode(t *testing.T) {
	got := fromFlags(Flags{Mode: string(ModeYolo)}).Mode
	if got != ModeYolo {
		t.Fatalf("mode = %q, want %q", got, ModeYolo)
	}
}
