package settings

import (
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	appsettings "github.com/vitaliiPsl/crappy-ai/internal/settings"
)

func TestModelOptionsUsesProviderName(t *testing.T) {
	m := Model{
		cfg: config.Config{Agent: config.Agent{Provider: "local", Model: "gpt-local"}},
		settings: appsettings.Settings{
			Providers: []appsettings.ProviderSettings{
				{ID: "local", API: "openai"},
			},
			Models: map[string][]kit.ModelConfig{
				"local":  {{ID: "gpt-local"}},
				"openai": {{ID: "gpt-5"}},
			},
		},
	}

	got := m.modelOptions()
	if len(got) != 1 || got[0].ID != "gpt-local" {
		t.Fatalf("modelOptions = %+v, want local provider catalog", got)
	}
}

func TestLabelCellDoesNotWrapLongestFieldLabel(t *testing.T) {
	cell := labelCell(maxOutputTokensLabel)
	if strings.Contains(cell, "\n") {
		t.Fatalf("labelCell(%q) wrapped:\n%s", maxOutputTokensLabel, cell)
	}
}

func TestProviderCredentialFieldsFollowAuthType(t *testing.T) {
	tests := []struct {
		name   string
		auth   appsettings.ProviderAuthType
		want   []string
		reject []string
	}{
		{
			name:   "api key",
			auth:   appsettings.ProviderAuthAPIKey,
			want:   []string{authTypeLabel, apiKeyLabel, apiKeyEnvLabel},
			reject: []string{oauthLabel},
		},
		{
			name: "oauth",
			auth: appsettings.ProviderAuthOAuth,
			want: []string{
				authTypeLabel,
				oauthDriverLabel,
				oauthClientIDLabel,
				oauthAuthURLLabel,
				oauthTokenURLLabel,
				oauthRedirectURLLabel,
				oauthScopesLabel,
				oauthLabel,
			},
			reject: []string{apiKeyLabel, apiKeyEnvLabel},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				cfg: config.Config{Agent: config.Agent{Provider: "openai"}},
				settings: appsettings.Settings{Providers: []appsettings.ProviderSettings{{
					ID:   "openai",
					API:  "openai",
					Auth: appsettings.ProviderAuthSettings{Type: tt.auth},
				}}},
			}
			m.fields = newFieldsModel(nil)
			m.refreshContent()

			for _, field := range m.fields.defs {
				if field.section == providerSection {
					if field.label != baseURLLabel {
						t.Fatalf("first provider field = %q, want %q", field.label, baseURLLabel)
					}

					break
				}
			}

			labels := make(map[string]bool, len(m.fields.defs))
			for _, field := range m.fields.defs {
				labels[field.label] = true
			}

			for _, label := range tt.want {
				if !labels[label] {
					t.Errorf("field %q is missing", label)
				}
			}

			for _, label := range tt.reject {
				if labels[label] {
					t.Errorf("field %q is visible", label)
				}
			}
		})
	}
}

func TestModelOptionsDoesNotFallBackToProviderAPI(t *testing.T) {
	m := Model{
		cfg: config.Config{Agent: config.Agent{Provider: "local", Model: "llama3.1:8b"}},
		settings: appsettings.Settings{
			Providers: []appsettings.ProviderSettings{
				{ID: "local", API: "openai"},
			},
			Models: map[string][]kit.ModelConfig{
				"openai": {{ID: "gpt-5"}},
			},
		},
	}

	if got := m.modelOptions(); got != nil {
		t.Fatalf("modelOptions = %+v, want nil without local provider catalog", got)
	}
}

func TestSetActiveProviderDefaultsUnknownModel(t *testing.T) {
	m := Model{
		cfg: config.Config{Agent: config.Agent{Provider: "openai", Model: "gpt-old"}},
		settings: appsettings.Settings{
			Providers: []appsettings.ProviderSettings{
				{ID: "google", API: "google"},
				{ID: "openai", API: "openai"},
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
		cfg: config.Config{Agent: config.Agent{Provider: "openai", Model: "shared-model"}},
		settings: appsettings.Settings{
			Providers: []appsettings.ProviderSettings{
				{ID: "local", API: "openai"},
				{ID: "openai", API: "openai"},
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

func TestPickModelIDUsesTypedValueWhenNoModelMatches(t *testing.T) {
	m := Model{
		modelPicker: newModelPicker([]kit.ModelConfig{
			{ID: "gpt-5.5"},
			{ID: "gpt-5.5-pro"},
		}),
	}

	m.modelPicker.SetModels([]kit.ModelConfig{
		{ID: "gpt-5.5"},
		{ID: "gpt-5.5-pro"},
	}, "")
	m.modelPicker.Update("llama3.1:8b")

	got, ok := m.pickModelID("llama3.1:8b")
	if !ok {
		t.Fatal("pickModelID returned ok=false, want true")
	}

	if got != "llama3.1:8b" {
		t.Fatalf("pickModelID = %q, want typed model id", got)
	}
}

func TestPickModelIDPrefersSelectedCatalogModel(t *testing.T) {
	m := Model{
		modelPicker: newModelPicker([]kit.ModelConfig{
			{ID: "gpt-5.5"},
		}),
	}

	m.modelPicker.SetModels([]kit.ModelConfig{
		{ID: "gpt-5.5"},
	}, "")
	m.modelPicker.Update("gpt")

	got, ok := m.pickModelID("gpt")
	if !ok {
		t.Fatal("pickModelID returned ok=false, want true")
	}

	if got != "gpt-5.5" {
		t.Fatalf("pickModelID = %q, want selected catalog model", got)
	}
}
