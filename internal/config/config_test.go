package config

import "testing"

func TestSubagentInheritsModelButNotPermissionsOrTools(t *testing.T) {
	cfg := Config{
		Agent: Agent{
			Provider: "google",
			Model:    "gemini-3.1-flash",
			Thinking: "medium",
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

	if got.Prompt != "Explore the codebase." {
		t.Fatalf("prompt = %q, want subagent's own", got.Prompt)
	}

	if len(got.Tools) != 1 || got.Tools[0] != "read_file" {
		t.Fatalf("tools = %v, want subagent's own allowlist", got.Tools)
	}
}

func TestSubagentOverridesInheritedModel(t *testing.T) {
	cfg := Config{
		Agent:  Agent{Provider: "google", Model: "gemini-3.1-flash"},
		Agents: []Agent{{Name: "heavy", Provider: "anthropic", Model: "claude-opus-4-8"}},
	}

	got, _ := cfg.Subagent("heavy")
	if got.Provider != "anthropic" || got.Model != "claude-opus-4-8" {
		t.Fatalf("override not applied: %+v", got)
	}
}

func TestSubagentUnknownReturnsFalse(t *testing.T) {
	cfg := Config{Agents: []Agent{{Name: "explorer"}}}
	if _, ok := cfg.Subagent("missing"); ok {
		t.Fatal("expected ok=false for unknown subagent")
	}
}
