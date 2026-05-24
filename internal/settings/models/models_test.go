package models

import (
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestMergeAddsConfiguredProviderModels(t *testing.T) {
	got := Merge(map[string][]kit.ModelConfig{
		"openai": {{ID: "gpt-5"}},
	}, map[string][]kit.ModelConfig{
		"openai-local": {{ID: "gemma4", ContextWindow: 131072}},
	})

	local := got["openai-local"]
	if len(local) != 1 {
		t.Fatalf("len(openai-local) = %d, want 1", len(local))
	}

	if local[0].Provider != "openai-local" {
		t.Fatalf("Provider = %q, want openai-local", local[0].Provider)
	}

	if local[0].InputLimit != 131072 {
		t.Fatalf("InputLimit = %d, want ContextWindow fallback", local[0].InputLimit)
	}

	if got["openai"][0].ID != "gpt-5" {
		t.Fatalf("base model changed: %+v", got["openai"])
	}
}

func TestMergeOverridesConfiguredModelsByID(t *testing.T) {
	got := Merge(map[string][]kit.ModelConfig{
		"openai-local": {
			{ID: "gemma4", ContextWindow: 32768},
			{ID: "llama3.1:8b", ContextWindow: 131072},
		},
	}, map[string][]kit.ModelConfig{
		"openai-local": {
			{ID: "gemma4", ContextWindow: 131072},
		},
	})

	if len(got["openai-local"]) != 2 {
		t.Fatalf("len(openai-local) = %d, want 2", len(got["openai-local"]))
	}

	if got["openai-local"][0].ContextWindow != 131072 {
		t.Fatalf("ContextWindow = %d, want configured override", got["openai-local"][0].ContextWindow)
	}

	if got["openai-local"][1].ID != "llama3.1:8b" {
		t.Fatalf("second model = %+v, want existing model preserved", got["openai-local"][1])
	}
}
