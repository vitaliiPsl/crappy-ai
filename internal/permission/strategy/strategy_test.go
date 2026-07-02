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
