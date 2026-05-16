package models

import (
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestFilterCatalog_IncludesAllUpstreamModelsForSupportedAPI(t *testing.T) {
	providers := []ProviderSettings{
		{
			Name: "anthropic",
			API:  "anthropic",
			Models: []kit.ModelConfig{
				{ID: "claude-opus-4-7"},
			},
		},
	}

	upstream := map[string]upstreamProvider{
		"anthropic": {Models: map[string]upstreamModel{
			"claude-opus-4-7":   {},
			"claude-sonnet-4-6": {},
			"claude-haiku-3-5":  {},
		}},
	}

	got := filterCatalog(providers, upstream)

	if len(got["anthropic"]) != 3 {
		t.Fatalf("len = %d, want 3 (all upstream models, not just curated)", len(got["anthropic"]))
	}
}

func TestFilterCatalog_PutsCuratedFirstThenByReleaseDate(t *testing.T) {
	providers := []ProviderSettings{
		{
			Name: "openai",
			API:  "openai",
			Models: []kit.ModelConfig{
				{ID: "gpt-5"},
				{ID: "gpt-5.4-mini"},
			},
		},
	}

	upstream := map[string]upstreamProvider{
		"openai": {Models: map[string]upstreamModel{
			"gpt-5.4":      {Release: "2026-03-05"},
			"gpt-5.4-mini": {Release: "2026-03-17"},
			"gpt-5":        {Release: "2025-08-07"},
			"gpt-3.5":      {Release: "2023-03-01"},
		}},
	}

	got := filterCatalog(providers, upstream)

	wantOrder := []string{
		// curated first, in defaults() order
		"gpt-5",
		"gpt-5.4-mini",
		// rest by release date desc, ID asc on tie
		"gpt-5.4",
		"gpt-3.5",
	}

	if len(got["openai"]) != len(wantOrder) {
		t.Fatalf("len = %d, want %d", len(got["openai"]), len(wantOrder))
	}

	for i, want := range wantOrder {
		if got["openai"][i].ID != want {
			t.Errorf("got[%d].ID = %q, want %q", i, got["openai"][i].ID, want)
		}
	}
}

func TestFilterCatalog_IncludesUpstreamForProviderWithNoCurated(t *testing.T) {
	providers := []ProviderSettings{
		{Name: "anthropic", API: "anthropic"},
	}

	upstream := map[string]upstreamProvider{
		"anthropic": {Models: map[string]upstreamModel{
			"claude-opus-4-7":  {},
			"claude-haiku-3-5": {},
		}},
	}

	got := filterCatalog(providers, upstream)
	if len(got["anthropic"]) != 2 {
		t.Fatalf("len = %d, want 2 (supported API → keep upstream even without curated baseline)", len(got["anthropic"]))
	}
}

func TestFilterCatalog_MapsAPIKeyToProviderName(t *testing.T) {
	providers := []ProviderSettings{
		{
			Name: "my-claude",
			API:  "anthropic",
			Models: []kit.ModelConfig{
				{ID: "claude-opus-4-7"},
			},
		},
	}

	upstream := map[string]upstreamProvider{
		"anthropic": {Models: map[string]upstreamModel{
			"claude-opus-4-7": {Limit: upstreamLimits{Context: 1_000_000}},
		}},
	}

	got := filterCatalog(providers, upstream)

	if _, ok := got["my-claude"]; !ok {
		t.Fatalf("output should be keyed by provider Name, got keys: %v", keys(got))
	}

	if _, ok := got["anthropic"]; ok {
		t.Errorf("output keyed by API, expected Name only")
	}
}

func TestFilterCatalog_SkipsUnknownUpstreamProvider(t *testing.T) {
	providers := []ProviderSettings{
		{
			Name: "azure-openai",
			API:  "azure",
			Models: []kit.ModelConfig{
				{ID: "gpt-5"},
			},
		},
	}

	upstream := map[string]upstreamProvider{
		"openai": {Models: map[string]upstreamModel{"gpt-5": {}}},
	}

	got := filterCatalog(providers, upstream)
	if len(got) != 0 {
		t.Fatalf("got %d providers, want 0 (api 'azure' has no upstream)", len(got))
	}
}

func TestNormalizeUpstreamModel_PopulatesCapabilitiesAndCost(t *testing.T) {
	u := upstreamModel{
		Reasoning: true,
		ToolCall:  true,
		Modalities: upstreamModes{
			Input:  []string{"text", "image"},
			Output: []string{"text"},
		},
		Cost: &upstreamCost{Input: 3, Output: 15, CacheRead: 0.3, CacheWrite: 3.75},
		Limit: upstreamLimits{
			Context: 1_000_000,
			Input:   1_000_000,
			Output:  64_000,
		},
		Knowledge: "2025-08-31",
		Release:   "2026-02-17T00:00:00Z",
	}

	got := normalizeUpstreamModel("anthropic", "claude-sonnet-4-6", u)

	if got.Provider != "anthropic" || got.ID != "claude-sonnet-4-6" {
		t.Fatalf("identity wrong: %+v", got)
	}

	if got.ContextWindow != 1_000_000 || got.InputLimit != 1_000_000 || got.OutputLimit != 64_000 {
		t.Fatalf("limits wrong: %+v", got)
	}

	if got.KnowledgeCutoff != "2025-08-31" {
		t.Errorf("KnowledgeCutoff = %q, want 2025-08-31", got.KnowledgeCutoff)
	}

	if got.ReleaseDate != "2026-02-17" {
		t.Errorf("ReleaseDate = %q, want 2026-02-17 (RFC3339 should be normalized)", got.ReleaseDate)
	}

	if !got.Capabilities.Text || !got.Capabilities.Image {
		t.Errorf("modalities not picked up: %+v", got.Capabilities)
	}

	if !got.Capabilities.Tools || !got.Capabilities.Reasoning || !got.Capabilities.Caching {
		t.Errorf("features not picked up: %+v", got.Capabilities)
	}

	if got.Cost.Input != 3 || got.Cost.CacheRead != 0.3 {
		t.Errorf("cost not populated: %+v", got.Cost)
	}
}

func TestNormalizeUpstreamModel_DefaultsInputLimitToContextWindow(t *testing.T) {
	u := upstreamModel{Limit: upstreamLimits{Context: 200_000}}

	got := normalizeUpstreamModel("anthropic", "claude-haiku-4-5", u)
	if got.InputLimit != 200_000 {
		t.Fatalf("InputLimit = %d, want 200_000 fallback to ContextWindow", got.InputLimit)
	}
}

func TestNormalizeUpstreamModel_CachingFalseWhenNoCachingPrice(t *testing.T) {
	u := upstreamModel{Cost: &upstreamCost{Input: 1, Output: 2}}

	got := normalizeUpstreamModel("openai", "gpt-x", u)
	if got.Capabilities.Caching {
		t.Errorf("Caching should be false when cache prices are zero")
	}
}

func keys(m map[string][]kit.ModelConfig) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}
