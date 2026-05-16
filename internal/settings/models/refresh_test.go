package models

import (
	"testing"
)

func TestNormalizeUpstreamSortsByReleaseDateDesc(t *testing.T) {
	upstream := map[string]upstreamProvider{
		"openai": {Models: map[string]upstreamModel{
			"gpt-old":     {Release: "2023-03-01"},
			"gpt-newest":  {Release: "2026-03-17"},
			"gpt-newer":   {Release: "2026-03-05"},
			"gpt-no-date": {},
			"gpt-tied":    {Release: "2026-03-05"},
		}},
	}

	got := normalizeUpstream(upstream)

	want := []string{"gpt-newest", "gpt-newer", "gpt-tied", "gpt-old", "gpt-no-date"}

	if len(got["openai"]) != len(want) {
		t.Fatalf("len = %d, want %d", len(got["openai"]), len(want))
	}

	for i, id := range want {
		if got["openai"][i].ID != id {
			t.Errorf("got[%d].ID = %q, want %q", i, got["openai"][i].ID, id)
		}
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

	got := mapUpstreamModel("anthropic", "claude-sonnet-4-6", u)

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

	got := mapUpstreamModel("anthropic", "claude-haiku-4-5", u)
	if got.InputLimit != 200_000 {
		t.Fatalf("InputLimit = %d, want 200_000 fallback to ContextWindow", got.InputLimit)
	}
}

func TestNormalizeUpstreamModel_CachingFalseWhenNoCachingPrice(t *testing.T) {
	u := upstreamModel{Cost: &upstreamCost{Input: 1, Output: 2}}

	got := mapUpstreamModel("openai", "gpt-x", u)
	if got.Capabilities.Caching {
		t.Errorf("Caching should be false when cache prices are zero")
	}
}
