package command_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/skills"
	"github.com/vitaliiPsl/crappy-ai/internal/skills/skillstest"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
)

func TestSkillCommandDefinitionMatchesSkill(t *testing.T) {
	cmd := command.NewSkillCommand(skills.Skill{Name: "review", Description: "Review code"})

	def := cmd.Definition()
	if def.Name != "review" {
		t.Fatalf("Name = %q, want review", def.Name)
	}

	if def.Description != "Review code" {
		t.Fatalf("Description = %q, want Review code", def.Description)
	}
}

func TestSkillCommandExecuteEmitsSkillRequest(t *testing.T) {
	cmd := command.NewSkillCommand(skills.Skill{Name: "review", Description: "Review code"})

	msg := cmd.Execute(context.Background(), command.Request{Raw: "/review auth changes", Args: []string{"auth", "changes"}})()

	submit, ok := msg.(command.SubmitSkillMsg)
	if !ok {
		t.Fatalf("msg = %#v, want SubmitSkillMsg", msg)
	}

	if submit.Text != "/review auth changes" {
		t.Fatalf("Text = %q, want raw command", submit.Text)
	}

	if submit.Name != "review" {
		t.Fatalf("Name = %q, want review", submit.Name)
	}

	if len(submit.Args) != 2 || submit.Args[0] != "auth" || submit.Args[1] != "changes" {
		t.Fatalf("Args = %#v, want auth changes", submit.Args)
	}
}

func TestRegistrySkipsSkillNameCollisionWithBuiltin(t *testing.T) {
	userDir := filepath.Join(t.TempDir(), "skills")
	skillstest.WriteSkill(t, filepath.Join(userDir, "help", "SKILL.md"), "help", "Custom help", "ignored")
	skillstest.WriteSkill(t, filepath.Join(userDir, "review", "SKILL.md"), "review", "Review code", "Find bugs first.")

	registry := command.NewRegistry(skillstest.NewRegistry(userDir))

	helpCmd, _ := registry.Get("help")
	if _, ok := helpCmd.(*command.HelpCommand); !ok {
		t.Fatalf("help = %T, want *command.HelpCommand (builtin must win)", helpCmd)
	}

	reviewCmd, ok := registry.Get("review")
	if !ok {
		t.Fatal("review command missing")
	}

	if _, ok := reviewCmd.(*command.SkillCommand); !ok {
		t.Fatalf("review = %T, want *command.SkillCommand", reviewCmd)
	}
}
