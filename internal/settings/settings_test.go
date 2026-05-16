package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/settings/models"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestMergeProvidersPreservesDefaultsForPartialOverride(t *testing.T) {
	base := Settings{
		Providers: []models.ProviderSettings{
			{
				Name: models.ProviderAnthropic,
				API:  models.ProviderAnthropic,
			},
			{
				Name:      models.ProviderOpenAI,
				API:       models.ProviderOpenAI,
				APIKeyEnv: "OPENAI_API_KEY",
				Models:    []kit.ModelConfig{{ID: "gpt-5"}},
			},
			{
				Name: models.ProviderGoogle,
				API:  models.ProviderGoogle,
			},
		},
	}
	overlay := Settings{
		Providers: []models.ProviderSettings{
			{
				Name:   models.ProviderOpenAI,
				APIKey: "secret",
			},
			{
				Name:    "local",
				API:     models.ProviderOpenAI,
				BaseURL: "http://localhost:11434",
			},
		},
	}

	got := merge(base, overlay)

	if len(got.Providers) != 4 {
		t.Fatalf("len(Providers) = %d, want 4", len(got.Providers))
	}

	if got.Providers[0].Name != models.ProviderAnthropic ||
		got.Providers[1].Name != models.ProviderGoogle ||
		got.Providers[2].Name != "local" ||
		got.Providers[3].Name != models.ProviderOpenAI {
		t.Fatalf("provider order = %q, %q, %q, %q; want anthropic, google, local, openai",
			got.Providers[0].Name,
			got.Providers[1].Name,
			got.Providers[2].Name,
			got.Providers[3].Name,
		)
	}

	openai := got.Providers[3]
	if openai.API != models.ProviderOpenAI {
		t.Errorf("API = %q, want %q", openai.API, models.ProviderOpenAI)
	}

	if openai.APIKeyEnv != "OPENAI_API_KEY" {
		t.Errorf("APIKeyEnv = %q, want OPENAI_API_KEY", openai.APIKeyEnv)
	}

	if openai.APIKey != "secret" {
		t.Errorf("APIKey = %q, want secret", openai.APIKey)
	}

	if len(openai.Models) != 1 || openai.Models[0].ID != "gpt-5" {
		t.Fatalf("Models = %+v, want gpt-5 metadata preserved", openai.Models)
	}
}

func TestStoreGetReturnsDeepCopy(t *testing.T) {
	store := NewStore(Settings{
		Providers: []models.ProviderSettings{
			{
				Name:   models.ProviderOpenAI,
				Models: []kit.ModelConfig{{ID: "gpt-5"}},
			},
		},
	}, "")

	got := store.Get()
	got.Providers[0].Name = "changed"
	got.Providers[0].Models[0].ID = "changed"

	again := store.Get()
	if again.Providers[0].Name != models.ProviderOpenAI {
		t.Fatalf("stored provider name = %q, want %q", again.Providers[0].Name, models.ProviderOpenAI)
	}

	if again.Providers[0].Models[0].ID != "gpt-5" {
		t.Fatalf("stored model ID = %q, want gpt-5", again.Providers[0].Models[0].ID)
	}
}

func TestWriteFileUsesPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.yaml")

	if err := writeFile(path, Settings{
		Providers: []models.ProviderSettings{
			{Name: models.ProviderOpenAI, APIKey: "secret"},
		},
	}); err != nil {
		t.Fatalf("writeFile() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("mode = %o, want 600", mode)
	}
}
