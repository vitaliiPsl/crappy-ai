package command_test

import (
	"context"
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
)

type promptSource struct {
	prompts []mcp.ServerPrompt
}

func (s *promptSource) GetMCPPrompts(context.Context) []mcp.ServerPrompt {
	return s.prompts
}

func TestMCPPromptCommandExecuteSubmitsInvocation(t *testing.T) {
	src := &promptSource{}
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

	submit, ok := msg.(command.SubmitMCPPromptMsg)
	if !ok {
		t.Fatalf("msg = %#v, want SubmitMCPPromptMsg", msg)
	}

	if submit.Server != "github" || submit.Name != "review" || submit.Args["path"] != "main.go" {
		t.Fatalf("submit = %+v, want github review path=main.go", submit)
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
	src := &promptSource{}
	cmd := command.NewMCPPromptCommand(src, mcp.ServerPrompt{
		Server: "everything",
		Prompt: mcp.Prompt{Name: "greet"},
	})

	msg := cmd.Execute(context.Background(), command.Request{Args: []string{"name=Vitalii"}})()

	submit, ok := msg.(command.SubmitMCPPromptMsg)
	if !ok {
		t.Fatalf("msg = %#v, want SubmitMCPPromptMsg", msg)
	}

	if submit.Args["name"] != "Vitalii" {
		t.Fatalf("args = %#v, want name=Vitalii", submit.Args)
	}
}

func TestMCPPromptCommandExplicitArgumentsCanSatisfyDeclaredArgs(t *testing.T) {
	src := &promptSource{}
	cmd := command.NewMCPPromptCommand(src, mcp.ServerPrompt{
		Server: "github",
		Prompt: mcp.Prompt{
			Name:      "review",
			Arguments: []mcp.PromptArgument{{Name: "path", Required: true}},
		},
	})

	msg := cmd.Execute(context.Background(), command.Request{Args: []string{"path=main.go"}})()

	submit, ok := msg.(command.SubmitMCPPromptMsg)
	if !ok {
		t.Fatalf("msg = %#v, want SubmitMCPPromptMsg", msg)
	}

	if submit.Args["path"] != "main.go" {
		t.Fatalf("args = %#v, want path=main.go", submit.Args)
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

	registry := command.NewRegistry(context.Background(), command.NewMCPPromptProvider(src))

	cmd, ok := registry.Get("mcp:github:review")
	if !ok {
		t.Fatal("mcp prompt command missing")
	}

	if _, ok := cmd.(*command.MCPPromptCommand); !ok {
		t.Fatalf("command = %T, want *MCPPromptCommand", cmd)
	}
}
