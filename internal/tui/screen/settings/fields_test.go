package settings

import (
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	appsettings "github.com/vitaliiPsl/crappy-ai/internal/settings"
)

func TestModelOptionsFallsBackToProviderAPI(t *testing.T) {
	m := Model{
		cfg: config.Config{Provider: "local", Model: "gpt-local"},
		settings: appsettings.Settings{
			Providers: []appsettings.ProviderSettings{
				{Name: "local", API: "openai"},
			},
			Models: map[string][]kit.ModelConfig{
				"openai": {{ID: "gpt-5"}},
			},
		},
	}

	got := m.modelOptions()
	if len(got) != 1 || got[0].ID != "gpt-5" {
		t.Fatalf("modelOptions = %+v, want openai catalog fallback", got)
	}
}

func TestSetActiveProviderDefaultsUnknownModel(t *testing.T) {
	m := Model{
		cfg: config.Config{Provider: "openai", Model: "gpt-old"},
		settings: appsettings.Settings{
			Providers: []appsettings.ProviderSettings{
				{Name: "google", API: "google"},
				{Name: "openai", API: "openai"},
			},
			Models: map[string][]kit.ModelConfig{
				"google": {{ID: "gemini-new"}},
				"openai": {{ID: "gpt-old"}},
			},
		},
	}

	m.setActiveProvider("google")

	if m.cfg.Provider != "google" {
		t.Fatalf("Provider = %q, want google", m.cfg.Provider)
	}

	if m.cfg.Model != "gemini-new" {
		t.Fatalf("Model = %q, want first google model", m.cfg.Model)
	}
}

func TestSetActiveProviderPreservesKnownModel(t *testing.T) {
	m := Model{
		cfg: config.Config{Provider: "openai", Model: "shared-model"},
		settings: appsettings.Settings{
			Providers: []appsettings.ProviderSettings{
				{Name: "local", API: "openai"},
				{Name: "openai", API: "openai"},
			},
			Models: map[string][]kit.ModelConfig{
				"openai": {{ID: "shared-model"}},
			},
		},
	}

	m.setActiveProvider("local")

	if m.cfg.Model != "shared-model" {
		t.Fatalf("Model = %q, want shared-model preserved", m.cfg.Model)
	}
}
