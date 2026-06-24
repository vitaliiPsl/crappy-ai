package config

import "testing"

func TestSubagentInheritsModelButNotPermissionsOrTools(t *testing.T) {
	temperature := float32(0.2)
	maxOutputTokens := int32(4096)

	cfg := Config{
		Agent: Agent{
			Provider:        "google",
			Model:           "gemini-3.1-flash",
			Thinking:        "medium",
			Temperature:     &temperature,
			MaxOutputTokens: &maxOutputTokens,
		},
		Agents: []Agent{
			{Name: "explorer", Prompt: "Explore the codebase.", Tools: []string{"read_file"}},
		},
	}

	got, ok := cfg.Subagent("explorer")
	if !ok {
		t.Fatal("Subagent(explorer) not found")
	}

	if got.Provider != "google" || got.Model != "gemini-3.1-flash" || got.Thinking != "medium" {
		t.Fatalf("model/thinking not inherited: %+v", got)
	}

	if got.Temperature == nil || *got.Temperature != temperature {
		t.Fatalf("temperature = %v, want inherited %v", got.Temperature, temperature)
	}

	if got.MaxOutputTokens == nil || *got.MaxOutputTokens != maxOutputTokens {
		t.Fatalf("max output tokens = %v, want inherited %v", got.MaxOutputTokens, maxOutputTokens)
	}

	if got.Prompt != "Explore the codebase." {
		t.Fatalf("prompt = %q, want subagent's own", got.Prompt)
	}

	if len(got.Tools) != 1 || got.Tools[0] != "read_file" {
		t.Fatalf("tools = %v, want subagent's own allowlist", got.Tools)
	}
}

func TestSubagentOverridesInheritedModel(t *testing.T) {
	temperature := float32(0.8)
	maxOutputTokens := int32(1024)

	cfg := Config{
		Agent: Agent{Provider: "google", Model: "gemini-3.1-flash"},
		Agents: []Agent{{
			Name:            "heavy",
			Provider:        "anthropic",
			Model:           "claude-opus-4-8",
			Temperature:     &temperature,
			MaxOutputTokens: &maxOutputTokens,
		}},
	}

	got, _ := cfg.Subagent("heavy")
	if got.Provider != "anthropic" || got.Model != "claude-opus-4-8" {
		t.Fatalf("override not applied: %+v", got)
	}

	if got.Temperature == nil || *got.Temperature != temperature {
		t.Fatalf("temperature override not applied: %+v", got)
	}

	if got.MaxOutputTokens == nil || *got.MaxOutputTokens != maxOutputTokens {
		t.Fatalf("max output override not applied: %+v", got)
	}
}

func TestSubagentUnknownReturnsFalse(t *testing.T) {
	cfg := Config{Agents: []Agent{{Name: "explorer"}}}
	if _, ok := cfg.Subagent("missing"); ok {
		t.Fatal("expected ok=false for unknown subagent")
	}
}
