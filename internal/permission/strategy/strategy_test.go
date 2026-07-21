package strategy

import (
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
)

func prompt(t *testing.T, call kit.ToolCall) model.Prompt {
	t.Helper()

	result := Resolve(model.Permissions{Default: model.Ask}, call)
	if result.Prompt == nil {
		t.Fatalf("Prompt = nil, want permission prompt")
	}

	return *result.Prompt
}

func optionRule(t *testing.T, request model.Prompt, id string) model.Rule {
	t.Helper()

	option, ok := request.Option(id)
	if !ok {
		t.Fatalf("option %q not found in %#v", id, request.Options)
	}

	if option.Rule == nil {
		t.Fatalf("option %q rule = nil", id)
	}

	return *option.Rule
}

func TestMemoryListIsAllowedUnlessExplicitlyDenied(t *testing.T) {
	call := kit.NewToolCall("call-1", ToolMemoryList, map[string]any{})

	result := Resolve(model.Permissions{Default: model.Ask}, call)
	if result.Decision != model.Allow {
		t.Fatalf("Resolve() decision = %q, want allow", result.Decision)
	}

	result = Resolve(model.Permissions{
		Default: model.Ask,
		Deny:    []model.Rule{{Tool: ToolMemoryList}},
	}, call)
	if result.Decision != model.Deny {
		t.Fatalf("Resolve() explicit deny decision = %q, want deny", result.Decision)
	}
}

func TestMemoryMutationUsesDefaultPermissionAndContentDetail(t *testing.T) {
	call := kit.NewToolCall("call-1", ToolMemoryRemember, map[string]any{
		"kind":    "preference",
		"content": "Prefers concise answers.",
	})

	result := Resolve(model.Permissions{Default: model.Ask}, call)
	if result.Decision != model.Ask || result.Prompt == nil {
		t.Fatalf("Resolve() = %+v, want ask prompt", result)
	}

	if result.Prompt.Input != "Prefers concise answers." {
		t.Fatalf("Prompt input = %q", result.Prompt.Input)
	}
}
