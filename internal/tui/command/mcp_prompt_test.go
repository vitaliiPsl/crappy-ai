package command_test

import (
	"context"
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/skills"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
)

type promptSource struct {
	prompts []mcp.ServerPrompt
	result  mcp.PromptResult
	got     struct {
		server string
		name   string
		args   map[string]string
	}
}

func (s *promptSource) GetSkills() []skills.Skill {
	return nil
}

func (s *promptSource) GetMCPPrompts(context.Context) []mcp.ServerPrompt {
	return s.prompts
}

func (s *promptSource) GetMCPPrompt(_ context.Context, server, name string, args map[string]string) (mcp.PromptResult, error) {
	s.got.server = server
	s.got.name = name
	s.got.args = args

	return s.result, nil
}

func TestMCPPromptCommandExecuteSubmitsResolvedPrompt(t *testing.T) {
	src := &promptSource{
		result: mcp.PromptResult{
			Messages: []mcp.PromptMessage{{
				Role: "user",
				Content: []mcp.PromptContent{{
					Type: "text",
					Text: "Review main.go",
				}},
			}},
		},
	}
	prompt := mcp.ServerPrompt{
		Server: "github",
		Prompt: mcp.Prompt{
			Name: "review",
			Arguments: []mcp.PromptArgument{{
				Name:     "path",
				Required: true,
			}},
		},
	}
	cmd := command.NewMCPPromptCommand(src, prompt)

	msg := cmd.Execute(context.Background(), command.Request{Args: []string{"main.go"}})()

	submit, ok := msg.(command.SubmitTextMsg)
	if !ok {
		t.Fatalf("msg = %#v, want SubmitTextMsg", msg)
	}

	if submit.Text != "Review main.go" {
		t.Fatalf("Text = %q, want resolved prompt text", submit.Text)
	}

	if src.got.server != "github" || src.got.name != "review" || src.got.args["path"] != "main.go" {
		t.Fatalf("GetMCPPrompt call = %+v, want github review path=main.go", src.got)
	}
}

func TestMCPPromptCommandMissingRequiredArgument(t *testing.T) {
	cmd := command.NewMCPPromptCommand(&promptSource{}, mcp.ServerPrompt{
		Server: "github",
		Prompt: mcp.Prompt{
			Name:      "review",
			Arguments: []mcp.PromptArgument{{Name: "path", Required: true}},
		},
	})

	msg := cmd.Execute(context.Background(), command.Request{})()

	if _, ok := msg.(command.SystemMsg); !ok {
		t.Fatalf("msg = %#v, want SystemMsg", msg)
	}
}

func TestMCPPromptCommandAcceptsExplicitArguments(t *testing.T) {
	src := &promptSource{
		result: mcp.PromptResult{
			Messages: []mcp.PromptMessage{{
				Content: []mcp.PromptContent{{Type: "text", Text: "Say hi to Vitalii"}},
			}},
		},
	}
	cmd := command.NewMCPPromptCommand(src, mcp.ServerPrompt{
		Server: "everything",
		Prompt: mcp.Prompt{Name: "greet"},
	})

	msg := cmd.Execute(context.Background(), command.Request{Args: []string{"name=Vitalii"}})()

	if _, ok := msg.(command.SubmitTextMsg); !ok {
		t.Fatalf("msg = %#v, want SubmitTextMsg", msg)
	}

	if src.got.args["name"] != "Vitalii" {
		t.Fatalf("args = %#v, want name=Vitalii", src.got.args)
	}
}

func TestMCPPromptCommandExplicitArgumentsCanSatisfyDeclaredArgs(t *testing.T) {
	src := &promptSource{
		result: mcp.PromptResult{
			Messages: []mcp.PromptMessage{{
				Content: []mcp.PromptContent{{Type: "text", Text: "Review main.go"}},
			}},
		},
	}
	cmd := command.NewMCPPromptCommand(src, mcp.ServerPrompt{
		Server: "github",
		Prompt: mcp.Prompt{
			Name:      "review",
			Arguments: []mcp.PromptArgument{{Name: "path", Required: true}},
		},
	})

	msg := cmd.Execute(context.Background(), command.Request{Args: []string{"path=main.go"}})()

	if _, ok := msg.(command.SubmitTextMsg); !ok {
		t.Fatalf("msg = %#v, want SubmitTextMsg", msg)
	}

	if src.got.args["path"] != "main.go" {
		t.Fatalf("args = %#v, want path=main.go", src.got.args)
	}
}

func TestRegistryIncludesMCPPromptCommands(t *testing.T) {
	src := &promptSource{
		prompts: []mcp.ServerPrompt{{
			Server: "github",
			Prompt: mcp.Prompt{
				Name:        "review",
				Description: "Review a thing",
			},
		}},
	}

	registry := command.NewRegistry(src)

	cmd, ok := registry.Get("mcp:github:review")
	if !ok {
		t.Fatal("mcp prompt command missing")
	}

	if _, ok := cmd.(*command.MCPPromptCommand); !ok {
		t.Fatalf("command = %T, want *MCPPromptCommand", cmd)
	}
}
