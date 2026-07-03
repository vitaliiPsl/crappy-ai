package session

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
)

func TestRenderPromptUsesPermissionToolData(t *testing.T) {
	call := kit.NewToolCall("call-1", "bash", map[string]any{"command": "go test ./..."})
	prompt := model.NewPrompt(call, "go test ./...", nil)

	got := renderPrompt(&prompt.Request, 120)
	if !strings.Contains(got, "Allow bash: $ go test ./...?") {
		t.Fatalf("prompt = %q, want command label", got)
	}
}

func TestRenderPromptHintsUsesPermissionRulePattern(t *testing.T) {
	call := kit.NewToolCall("call-1", "read_file", map[string]any{"path": "/tmp/project/main.go"})
	prompt := model.NewPrompt(call, "/tmp/project/main.go", []model.Option{
		{
			ID:       model.OptionAllowPattern,
			Label:    "Allow project",
			Decision: model.Allow,
			Scope:    model.ScopeGlobal,
			Rule:     &model.Rule{Tool: "read_file", Pattern: "//tmp/project/**"},
		},
	})

	got := renderPromptHints(&prompt.Request, 120)
	if !strings.Contains(got, "g Pattern: //tmp/project/**") {
		t.Fatalf("hints = %q, want rule pattern", got)
	}
}

func TestPickPromptOptionUsesPermissionData(t *testing.T) {
	call := kit.NewToolCall("call-1", "read_file", map[string]any{"path": "/tmp/project/main.go"})
	prompt := model.NewPrompt(call, "/tmp/project/main.go", []model.Option{
		{
			ID:       model.OptionAllowPattern,
			Label:    "Allow project",
			Decision: model.Allow,
			Scope:    model.ScopeGlobal,
			Rule:     &model.Rule{Tool: "read_file", Pattern: "//tmp/project/**"},
		},
	})

	got := pickPromptOption(tea.KeyPressMsg(tea.Key{Text: "g", Code: 'g'}), prompt.Request)
	if got != model.OptionAllowPattern {
		t.Fatalf("option = %q, want %q", got, model.OptionAllowPattern)
	}
}
