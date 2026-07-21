package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
	"github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

func TestMergeProvidersPreservesDefaultsForPartialOverride(t *testing.T) {
	base := Settings{
		Providers: []ProviderSettings{
			{
				ID:  models.ProviderAnthropic,
				API: models.ProviderAnthropic,
			},
			{
				ID:  models.ProviderOpenAI,
				API: models.ProviderOpenAI,
				Auth: ProviderAuthSettings{
					Type:      ProviderAuthAPIKey,
					APIKeyEnv: "OPENAI_API_KEY",
				},
			},
			{
				ID:  models.ProviderGoogle,
				API: models.ProviderGoogle,
			},
		},
		Models: map[string][]kit.ModelConfig{
			models.ProviderOpenAI: {{ID: "gpt-5"}},
		},
		ModelConfigs: map[string][]kit.ModelConfig{
			"openai-local": {{ID: "llama3.1:8b"}},
		},
	}
	overlay := Settings{
		Providers: []ProviderSettings{
			{
				ID: models.ProviderOpenAI,
				Auth: ProviderAuthSettings{
					Type:   ProviderAuthAPIKey,
					APIKey: "secret",
				},
			},
			{
				ID:      "local",
				API:     models.ProviderOpenAI,
				BaseURL: "http://localhost:11434",
			},
		},
		ModelConfigs: map[string][]kit.ModelConfig{
			"openai-local": {{ID: "gemma4"}},
		},
	}

	got := merge(base, overlay)

	if len(got.Providers) != 4 {
		t.Fatalf("len(Providers) = %d, want 4", len(got.Providers))
	}

	if got.Providers[0].ID != models.ProviderAnthropic ||
		got.Providers[1].ID != models.ProviderGoogle ||
		got.Providers[2].ID != "local" ||
		got.Providers[3].ID != models.ProviderOpenAI {
		t.Fatalf("provider order = %q, %q, %q, %q; want anthropic, google, local, openai",
			got.Providers[0].ID,
			got.Providers[1].ID,
			got.Providers[2].ID,
			got.Providers[3].ID,
		)
	}

	openai := got.Providers[3]
	if openai.API != models.ProviderOpenAI {
		t.Errorf("API = %q, want %q", openai.API, models.ProviderOpenAI)
	}

	if openai.Auth.APIKeyEnv != "" {
		t.Errorf("APIKeyEnv = %q, want empty after auth replacement", openai.Auth.APIKeyEnv)
	}

	if openai.Auth.APIKey != "secret" {
		t.Errorf("APIKey = %q, want secret", openai.Auth.APIKey)
	}

	if len(got.Models[models.ProviderOpenAI]) != 1 || got.Models[models.ProviderOpenAI][0].ID != "gpt-5" {
		t.Fatalf("Models = %+v, want gpt-5 metadata preserved", got.Models[models.ProviderOpenAI])
	}

	if len(got.ModelConfigs["openai-local"]) != 2 {
		t.Fatalf("ModelConfigs = %+v, want merged local models", got.ModelConfigs["openai-local"])
	}
}

func TestMergeReadsSkillsPath(t *testing.T) {
	got := merge(Settings{SkillsPath: "/default/skills"}, Settings{SkillsPath: "/custom/skills"})

	if got.SkillsPath != "/custom/skills" {
		t.Fatalf("SkillsPath = %q, want /custom/skills", got.SkillsPath)
	}
}

func TestMergeReadsMemoryPath(t *testing.T) {
	got := merge(Settings{MemoryPath: "/default/memory.json"}, Settings{MemoryPath: "/custom/memory.json"})

	if got.MemoryPath != "/custom/memory.json" {
		t.Fatalf("MemoryPath = %q, want /custom/memory.json", got.MemoryPath)
	}
}

func TestMergeReadsMCPClients(t *testing.T) {
	got := merge(Settings{}, Settings{
		MCPClients: []mcp.Config{{
			Name:    "github",
			Command: "github-mcp-server",
		}},
	})

	if len(got.MCPClients) != 1 {
		t.Fatalf("len(MCPClients) = %d, want 1", len(got.MCPClients))
	}

	if got.MCPClients[0].Name != "github" || got.MCPClients[0].Command != "github-mcp-server" {
		t.Fatalf("MCPClients[0] = %+v, want github config", got.MCPClients[0])
	}
}

func TestWriteFileUsesPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.yaml")

	if err := writeFile(path, Settings{
		Providers: []ProviderSettings{
			{
				ID:   models.ProviderOpenAI,
				Auth: ProviderAuthSettings{Type: ProviderAuthAPIKey, APIKey: "secret"},
			},
		},
		ModelConfigs: map[string][]kit.ModelConfig{
			"openai-local": {{ID: "gemma4"}},
		},
		Models: map[string][]kit.ModelConfig{
			models.ProviderOpenAI: {{ID: "gpt-5"}},
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

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(data) == "" || strings.Contains(string(data), "gpt-5") {
		t.Fatalf("settings file should not persist available models, got:\n%s", data)
	}

	if !strings.Contains(string(data), "gemma4") {
		t.Fatalf("settings file should persist configured models, got:\n%s", data)
	}
}

func TestProviderOAuthConfigIsInlineInYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.yaml")
	want := provideroauth.Config{
		ClientID:         "client",
		AuthorizationURL: "https://auth.example/authorize",
		TokenURL:         "https://auth.example/token",
		RedirectURL:      "http://localhost/callback",
		Scopes:           []string{"openid", "offline_access"},
		LimitsURL:        "https://example.test/usage",
	}

	err := writeFile(path, Settings{Providers: []ProviderSettings{{
		ID:  "subscription",
		API: "openai",
		Auth: ProviderAuthSettings{
			Type:   ProviderAuthOAuth,
			Driver: "openai-codex",
			OAuth:  want,
		},
	}}})
	if err != nil {
		t.Fatalf("writeFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if strings.Contains(string(data), "oauth:") || !strings.Contains(string(data), "client_id: client") {
		t.Fatalf("OAuth config is not inline:\n%s", data)
	}

	got, _, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile() error = %v", err)
	}

	if len(got.Providers) != 1 || got.Providers[0].Auth.OAuth.ClientID != want.ClientID ||
		got.Providers[0].Auth.OAuth.TokenURL != want.TokenURL ||
		got.Providers[0].Auth.OAuth.LimitsURL != want.LimitsURL {
		t.Fatalf("OAuth config = %+v, want %+v", got.Providers, want)
	}
}

func TestLoadMergesConfiguredModelsIntoCatalog(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.yaml")

	data := []byte(`
models_path: ` + filepath.Join(dir, "models.json") + `
providers:
  - id: openai-local
    api: openai
    base_url: http://localhost:11434/v1
    auth:
      type: api_key
      api_key: local
models:
  openai-local:
    - id: gemma4
      context_window: 131072
      output_limit: 8192
      capabilities:
        text: true
        tools: true
        streaming: true
`)

	if err := os.WriteFile(settingsPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv(EnvSettingsPath, settingsPath)
	t.Setenv(EnvModelsPath, "")
	t.Setenv(EnvSessionsDir, "")
	t.Setenv(EnvSkillsPath, "")

	store, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := store.Get().Models["openai-local"]
	if len(got) != 1 {
		t.Fatalf("len(Models[openai-local]) = %d, want 1", len(got))
	}

	model := got[0]
	if model.ID != "gemma4" || model.Provider != "openai-local" {
		t.Fatalf("model identity = %+v, want gemma4 for openai-local", model)
	}

	if model.ContextWindow != 131072 || model.InputLimit != 131072 || model.OutputLimit != 8192 {
		t.Fatalf("model limits = %+v", model)
	}

	if !model.Capabilities.Text || !model.Capabilities.Tools || !model.Capabilities.Streaming {
		t.Fatalf("capabilities = %+v", model.Capabilities)
	}
}
