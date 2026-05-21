package strategy

import (
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
)

func askRequest(t *testing.T, call kit.ToolCall) model.AskRequest {
	t.Helper()

	result := Resolve(model.Permissions{Default: model.Ask}, call)
	if result.AskRequest == nil {
		t.Fatalf("AskRequest = nil, want permission ask request")
	}

	return *result.AskRequest
}

func optionRule(t *testing.T, request model.AskRequest, id string) model.Rule {
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
