package models

import (
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
	settingsmodels "github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

func testSettings() settings.Settings {
	return settings.Settings{
		Providers: []settings.ProviderSettings{
			{Name: settingsmodels.ProviderAnthropic, API: settingsmodels.ProviderAnthropic, APIKey: "test-key"},
			{Name: settingsmodels.ProviderOpenAI, API: settingsmodels.ProviderOpenAI, APIKey: "test-key"},
			{Name: settingsmodels.ProviderGoogle, API: settingsmodels.ProviderGoogle, APIKey: "test-key"},
		},
		Models: map[string][]kit.ModelConfig{
			settingsmodels.ProviderOpenAI: {
				{ID: "gpt-5", ContextWindow: 400_000},
			},
		},
	}
}

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()

	return NewRegistry(settings.NewStore(testSettings(), ""))
}

func TestGetProviders(t *testing.T) {
	r := newTestRegistry(t)

	got := r.GetProviders()
	if len(got) != 3 {
		t.Errorf("GetProviders len = %d, want 3", len(got))
	}
}

func TestGetProvider(t *testing.T) {
	r := newTestRegistry(t)

	p, err := r.GetProvider(settingsmodels.ProviderAnthropic)
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}

	if p.Name != settingsmodels.ProviderAnthropic {
		t.Errorf("Name = %q, want %q", p.Name, settingsmodels.ProviderAnthropic)
	}

	if _, err := r.GetProvider("unknown"); err == nil {
		t.Error("GetProvider on unknown: expected error, got nil")
	}
}

func TestBuildModel_RequiresProvider(t *testing.T) {
	if _, err := buildModel(testSettings(), config.Config{Model: "x"}); err == nil {
		t.Fatal("expected error for empty provider")
	}
}

func TestBuildModel_RequiresModel(t *testing.T) {
	cfg := config.Config{Provider: settingsmodels.ProviderAnthropic}
	if _, err := buildModel(testSettings(), cfg); err == nil {
		t.Fatal("expected error for empty model")
	}
}

func TestBuildModel_UnknownProvider(t *testing.T) {
	cfg := config.Config{Provider: "mystery", Model: "m"}
	if _, err := buildModel(testSettings(), cfg); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestBuildModel_UnknownAPI(t *testing.T) {
	s := settings.Settings{
		Providers: []settings.ProviderSettings{
			{Name: "weird", API: "weird-api", APIKey: "k"},
		},
	}

	cfg := config.Config{Provider: "weird", Model: "m"}
	if _, err := buildModel(s, cfg); err == nil {
		t.Fatal("expected error for unknown api")
	}
}

func TestBuildModel_NoAPIKey(t *testing.T) {
	const envVar = "CRAPPY_TEST_NO_API_KEY"

	t.Setenv(envVar, "")

	s := settings.Settings{
		Providers: []settings.ProviderSettings{
			{Name: settingsmodels.ProviderAnthropic, API: settingsmodels.ProviderAnthropic, APIKeyEnv: envVar},
		},
	}

	cfg := config.Config{Provider: settingsmodels.ProviderAnthropic, Model: "claude-sonnet-4"}
	if _, err := buildModel(s, cfg); err == nil {
		t.Fatal("expected error for missing api key")
	}
}

func TestBuildModel_APIKeyFromEnv(t *testing.T) {
	const envVar = "CRAPPY_TEST_API_KEY"

	t.Setenv(envVar, "from-env")

	s := settings.Settings{
		Providers: []settings.ProviderSettings{
			{Name: settingsmodels.ProviderAnthropic, API: settingsmodels.ProviderAnthropic, APIKeyEnv: envVar},
		},
	}

	cfg := config.Config{Provider: settingsmodels.ProviderAnthropic, Model: "claude-sonnet-4"}

	m, err := buildModel(s, cfg)
	if err != nil {
		t.Fatalf("buildModel: %v", err)
	}

	if m == nil {
		t.Fatal("buildModel returned nil model")
	}
}

func TestBuildModel_UsesSettingsModels(t *testing.T) {
	s := testSettings()
	cfg := config.Config{Provider: settingsmodels.ProviderOpenAI, Model: "gpt-5"}

	m, err := buildModel(s, cfg)
	if err != nil {
		t.Fatalf("buildModel: %v", err)
	}

	if got := m.Config().ContextWindow; got != 400_000 {
		t.Fatalf("ContextWindow = %d, want metadata from Models", got)
	}
}

func TestBuildModel_DispatchesToEachProvider(t *testing.T) {
	cases := []struct {
		provider string
		model    string
	}{
		{settingsmodels.ProviderAnthropic, "claude-sonnet-4"},
		{settingsmodels.ProviderOpenAI, "gpt-5"},
		{settingsmodels.ProviderGoogle, "gemini-3-flash"},
	}

	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			cfg := config.Config{Provider: tc.provider, Model: tc.model}

			m, err := buildModel(testSettings(), cfg)
			if err != nil {
				t.Fatalf("buildModel: %v", err)
			}

			if m == nil {
				t.Fatal("buildModel returned nil model")
			}
		})
	}
}
