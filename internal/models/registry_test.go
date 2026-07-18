package models

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"
	adkproviders "github.com/vitaliiPsl/crappy-adk/providers"

	appproviders "github.com/vitaliiPsl/crappy-ai/internal/providers"
	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
	provideroauthstore "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth/store"
	"github.com/vitaliiPsl/crappy-ai/internal/providers/opencodex"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
	settingsmodels "github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

func testSettings() settings.Settings {
	return settings.Settings{
		Providers: []settings.ProviderSettings{
			{ID: settingsmodels.ProviderAnthropic, API: settingsmodels.ProviderAnthropic, Auth: settings.ProviderAuthSettings{Type: settings.ProviderAuthAPIKey, APIKey: "test-key"}},
			{ID: settingsmodels.ProviderOpenAI, API: settingsmodels.ProviderOpenAI, Auth: settings.ProviderAuthSettings{Type: settings.ProviderAuthAPIKey, APIKey: "test-key"}},
			{ID: settingsmodels.ProviderGoogle, API: settingsmodels.ProviderGoogle, Auth: settings.ProviderAuthSettings{Type: settings.ProviderAuthAPIKey, APIKey: "test-key"}},
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

	return NewRegistry(settings.NewStore(testSettings(), ""), nil)
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

	if p.ID != settingsmodels.ProviderAnthropic {
		t.Errorf("ID = %q, want %q", p.ID, settingsmodels.ProviderAnthropic)
	}

	if _, err := r.GetProvider("unknown"); err == nil {
		t.Error("GetProvider on unknown: expected error, got nil")
	}
}

func TestBuildModel_RequiresProvider(t *testing.T) {
	if _, err := buildTestModel(testSettings(), nil, "", "x"); err == nil {
		t.Fatal("expected error for empty provider")
	}
}

func TestBuildModel_RequiresModel(t *testing.T) {
	if _, err := buildTestModel(testSettings(), nil, settingsmodels.ProviderAnthropic, ""); err == nil {
		t.Fatal("expected error for empty model")
	}
}

func TestBuildModel_UnknownProvider(t *testing.T) {
	if _, err := buildTestModel(testSettings(), nil, "mystery", "m"); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestBuildModel_UnknownAPI(t *testing.T) {
	s := settings.Settings{
		Providers: []settings.ProviderSettings{
			{ID: "weird", API: "weird-api", Auth: settings.ProviderAuthSettings{Type: settings.ProviderAuthAPIKey, APIKey: "k"}},
		},
	}

	if _, err := buildTestModel(s, nil, "weird", "m"); err == nil {
		t.Fatal("expected error for unknown api")
	}
}

func TestBuildModel_NoAPIKey(t *testing.T) {
	const envVar = "CRAPPY_TEST_NO_API_KEY"

	t.Setenv(envVar, "")

	s := settings.Settings{
		Providers: []settings.ProviderSettings{
			{ID: settingsmodels.ProviderAnthropic, API: settingsmodels.ProviderAnthropic, Auth: settings.ProviderAuthSettings{Type: settings.ProviderAuthAPIKey, APIKeyEnv: envVar}},
		},
	}

	if _, err := buildTestModel(s, nil, settingsmodels.ProviderAnthropic, "claude-sonnet-4"); err == nil {
		t.Fatal("expected error for missing api key")
	}
}

func TestBuildModel_APIKeyFromEnv(t *testing.T) {
	const envVar = "CRAPPY_TEST_API_KEY"

	t.Setenv(envVar, "from-env")

	s := settings.Settings{
		Providers: []settings.ProviderSettings{
			{ID: settingsmodels.ProviderAnthropic, API: settingsmodels.ProviderAnthropic, Auth: settings.ProviderAuthSettings{Type: settings.ProviderAuthAPIKey, APIKeyEnv: envVar}},
		},
	}

	m, err := buildTestModel(s, nil, settingsmodels.ProviderAnthropic, "claude-sonnet-4")
	if err != nil {
		t.Fatalf("buildModel: %v", err)
	}

	if m == nil {
		t.Fatal("buildModel returned nil model")
	}
}

func TestBuildModel_UsesSettingsModels(t *testing.T) {
	s := testSettings()

	m, err := buildTestModel(s, nil, settingsmodels.ProviderOpenAI, "gpt-5")
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
			m, err := buildTestModel(testSettings(), nil, tc.provider, tc.model)
			if err != nil {
				t.Fatalf("buildModel: %v", err)
			}

			if m == nil {
				t.Fatal("buildModel returned nil model")
			}
		})
	}
}

func TestBuildModelUsesOpenAIOAuthWithoutAPIKey(t *testing.T) {
	store, err := provideroauthstore.NewFileStore(filepath.Join(t.TempDir(), "oauth.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	if err := store.Save(context.Background(), "work-openai", provideroauth.Credential{
		AccessToken: "access",
		ExpiresAt:   time.Now().Add(time.Hour),
		Metadata:    map[string]string{"account_id": "account"},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	providerManager := appproviders.NewManager(store, nil, opencodex.New())

	s := testSettings()
	s.Providers[1].ID = "work-openai"
	s.Providers[1].Auth = settings.ProviderAuthSettings{
		Type:   settings.ProviderAuthOAuth,
		Driver: opencodex.DriverID,
	}
	s.Models["work-openai"] = s.Models[settingsmodels.ProviderOpenAI]

	var got adkproviders.ModelOptions

	original := apiAdapters[settingsmodels.ProviderOpenAI]
	apiAdapters[settingsmodels.ProviderOpenAI] = func(id string, opts ...adkproviders.ModelOption) (kit.Model, error) {
		for _, opt := range opts {
			opt(&got)
		}

		return original(id, opts...)
	}
	t.Cleanup(func() { apiAdapters[settingsmodels.ProviderOpenAI] = original })

	if _, err := buildTestModel(s, providerManager, "work-openai", "gpt-5.5"); err != nil {
		t.Fatalf("buildModel() error = %v", err)
	}

	if got.BaseURL != opencodex.CodexAPIURL || got.BearerToken != "access" {
		t.Fatalf("model options = %+v", got)
	}

	if got.Headers["ChatGPT-Account-Id"] != "account" {
		t.Fatalf("account header = %q, want account", got.Headers["ChatGPT-Account-Id"])
	}
}

func TestBuildModelOAuthRejectsAPIKeySettings(t *testing.T) {
	store, err := provideroauthstore.NewFileStore(filepath.Join(t.TempDir(), "oauth.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	providerManager := appproviders.NewManager(store, nil, opencodex.New())
	s := settings.Settings{
		Providers: []settings.ProviderSettings{{
			ID:  "work-openai",
			API: settingsmodels.ProviderOpenAI,
			Auth: settings.ProviderAuthSettings{
				Type:   settings.ProviderAuthOAuth,
				APIKey: "must-not-be-used",
				Driver: opencodex.DriverID,
			},
		}},
	}

	_, err = buildTestModel(s, providerManager, "work-openai", "gpt-5.5")
	if err == nil || !strings.Contains(err.Error(), "API key settings cannot be used with oauth") {
		t.Fatalf("buildModel() error = %v, want conflicting auth settings error", err)
	}
}

func buildTestModel(s settings.Settings, providerManager *appproviders.Manager, provider, model string) (kit.Model, error) {
	return NewRegistry(settings.NewStore(s, ""), providerManager).Build(context.Background(), provider, model)
}
