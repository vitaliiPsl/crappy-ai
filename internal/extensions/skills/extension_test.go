package skills

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/kittest"
	xmemory "github.com/vitaliiPsl/crappy-adk/x/memory"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
	coreskills "github.com/vitaliiPsl/crappy-ai/internal/skills"
	"github.com/vitaliiPsl/crappy-ai/internal/skills/skillstest"
)

func TestUseSkillToolLoadsFullInstructions(t *testing.T) {
	userDir := filepath.Join(t.TempDir(), "skills")
	skillstest.WriteSkill(t, filepath.Join(userDir, "review", "SKILL.md"), "review", "review skill", "Find bugs first.")
	registry := skillstest.NewRegistry(userDir)

	got, err := newTool(registry).Execute(kit.NewRunContext(t.Context()), map[string]any{
		"skill": "review",
		"args":  "auth changes",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	for _, want := range []string{
		"Loaded skill: review",
		"Arguments:\nauth changes",
		"# Skill: review",
		"## Instructions",
		"Find bugs first.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("tool output missing %q:\n%s", want, got)
		}
	}
}

func TestUseSkillToolUnknownSkill(t *testing.T) {
	userDir := filepath.Join(t.TempDir(), "skills")
	skillstest.WriteSkill(t, filepath.Join(userDir, "review", "SKILL.md"), "review", "review skill", "Review changes.")
	registry := skillstest.NewRegistry(userDir)

	_, err := newTool(registry).Execute(kit.NewRunContext(t.Context()), map[string]any{"skill": "missing"})
	if err == nil {
		t.Fatal("Execute error = nil, want unknown skill")
	}

	if !errors.Is(err, coreskills.ErrUnknownSkill) {
		t.Fatalf("error = %v, want ErrUnknownSkill", err)
	}

	if strings.Contains(err.Error(), "review") {
		t.Fatalf("error = %q, should not list available skills", err)
	}
}

func TestExtensionAddsMetadataListingAndTool(t *testing.T) {
	userDir := filepath.Join(t.TempDir(), "skills")
	skillstest.WriteSkill(t, filepath.Join(userDir, "review", "SKILL.md"), "review", "review skill", "Find bugs first.")
	registry := skillstest.NewRegistry(userDir)

	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{
			Message:      kit.NewModelMessage(kit.NewTextContent("done")),
			FinishReason: kit.FinishReasonStop,
		},
	})

	ext := New(registry)

	extSpec, err := ext.Spec(factory.Context{})
	if err != nil {
		t.Fatalf("Spec: %v", err)
	}

	compiled, err := spec.Compile(extSpec)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	ag, err := agent.New(model, xmemory.NewHistory(), compiled.Tools, compiled.Options...)
	if err != nil {
		t.Fatalf("New agent: %v", err)
	}

	_, err = ag.Run(context.Background(), kit.NewUserMessage(kit.NewTextContent("review this")))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	req := model.CallAt(0)
	for _, want := range []string{
		"# Skills",
		"Available skills:",
		"- review: review skill",
	} {
		if !strings.Contains(req.Instructions, want) {
			t.Fatalf("instructions missing %q:\n%s", want, req.Instructions)
		}
	}

	if strings.Contains(req.Instructions, "Find bugs first.") {
		t.Fatalf("instructions leaked skill body:\n%s", req.Instructions)
	}

	if len(req.Tools) != 1 || req.Tools[0].Definition().Name != toolName {
		t.Fatalf("tools = %#v, want use_skill", req.Tools)
	}
}
