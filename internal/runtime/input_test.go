package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	mcpcore "github.com/vitaliiPsl/crappy-ai/internal/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/skills/skillstest"
)

func TestInputProcessorProcessesText(t *testing.T) {
	processor := NewInputProcessor("s1", nil, nil)

	msg, ev, err := processor.Process(context.Background(), Request{Text: "hello"})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	content := msg.TextContent()
	if content == nil || content.Text != "hello" {
		t.Fatalf("message text = %#v, want hello", content)
	}

	if ev.Message == nil || ev.Message.TextContent().Text != "hello" {
		t.Fatalf("event message = %+v, want text hello", ev.Message)
	}

	if ev.Skill != nil || ev.MCPPrompt != nil {
		t.Fatalf("event metadata skill=%+v mcp=%+v, want none", ev.Skill, ev.MCPPrompt)
	}
}

func TestInputProcessorProcessesSkill(t *testing.T) {
	root := t.TempDir()
	skillstest.WriteSkill(t, filepath.Join(root, "review", "SKILL.md"), "review", "Review code", "Read the diff carefully.")

	req := Request{
		Text: "ignored",
		Skill: &SkillInvocation{
			Name: "review",
			Args: []string{"file.go"},
		},
	}

	processor := NewInputProcessor("s1", skillstest.NewRegistry(root), nil)

	msg, ev, err := processor.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	content := msg.TextContent()
	if content == nil {
		t.Fatal("message has no text content")
	}

	for _, want := range []string{
		"Loaded skill: review",
		"Arguments:\nfile.go",
		"Read the diff carefully.",
	} {
		if !strings.Contains(content.Text, want) {
			t.Fatalf("message text missing %q:\n%s", want, content.Text)
		}
	}

	if ev.Skill == nil || ev.Skill.Name != "review" || strings.Join(ev.Skill.Args, " ") != "file.go" {
		t.Fatalf("skill event = %+v, want review invocation", ev.Skill)
	}
}

func TestInputProcessorSkillRequiresRegistry(t *testing.T) {
	processor := NewInputProcessor("s1", nil, nil)

	_, _, err := processor.Process(context.Background(), Request{
		Skill: &SkillInvocation{Name: "review"},
	})
	if err == nil || !strings.Contains(err.Error(), "skill registry is not configured") {
		t.Fatalf("Process error = %v, want missing skill registry", err)
	}
}

func TestInputProcessorProcessesMCPPrompt(t *testing.T) {
	manager := newPromptManager(t)
	defer manager.Close()

	req := Request{
		Text: "ignored",
		MCPPrompt: &MCPPromptInvocation{
			Server: "prompts",
			Name:   "review",
			Args:   map[string]string{"path": "main.go"},
		},
	}

	processor := NewInputProcessor("s1", nil, manager)

	msg, ev, err := processor.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	content := msg.TextContent()
	if content == nil {
		t.Fatal("message has no text content")
	}

	if content.Text != "Review main.go" {
		t.Fatalf("message text = %q, want resolved mcp prompt text", content.Text)
	}

	if ev.MCPPrompt == nil ||
		ev.MCPPrompt.Server != "prompts" ||
		ev.MCPPrompt.Name != "review" ||
		ev.MCPPrompt.Args["path"] != "main.go" {
		t.Fatalf("mcp prompt event = %+v, want prompts review path=main.go", ev.MCPPrompt)
	}
}

func TestInputProcessorMCPPromptRequiresManager(t *testing.T) {
	processor := NewInputProcessor("s1", nil, nil)

	_, _, err := processor.Process(context.Background(), Request{
		MCPPrompt: &MCPPromptInvocation{Server: "prompts", Name: "review"},
	})
	if err == nil || !strings.Contains(err.Error(), "mcp manager is not configured") {
		t.Fatalf("Process error = %v, want missing mcp manager", err)
	}
}

func TestFormatMCPPromptResultHandlesContentTypes(t *testing.T) {
	got := formatMCPPromptResult(mcpcore.PromptResult{
		Messages: []mcpcore.PromptMessage{{
			Content: []mcpcore.PromptContent{
				{Type: "text", Text: "hello"},
				{Type: "resource_link", URI: "file://README.md"},
				{Type: "image", MIMEType: "image/png", Data: []byte("abc")},
			},
		}},
	})

	for _, want := range []string{
		"hello",
		"[resource: file://README.md]",
		"[image: image/png, 3 bytes]",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatted prompt missing %q:\n%s", want, got)
		}
	}
}

func newPromptManager(t *testing.T) *mcpcore.Manager {
	t.Helper()

	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "prompts", Version: "0.1.0"}, nil)
	server.AddPrompt(&mcpsdk.Prompt{
		Name: "review",
		Arguments: []*mcpsdk.PromptArgument{{
			Name:     "path",
			Required: true,
		}},
	}, func(_ context.Context, req *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
		return &mcpsdk.GetPromptResult{
			Messages: []*mcpsdk.PromptMessage{{
				Role:    "user",
				Content: &mcpsdk.TextContent{Text: "Review " + req.Params.Arguments["path"]},
			}},
		}, nil
	})

	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return server
	}, nil)
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	manager := mcpcore.NewManager(
		context.Background(),
		[]mcpcore.Config{{Name: "prompts", Transport: mcpcore.TransportHTTP, URL: httpServer.URL}},
		nil,
		nil,
	)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		for _, client := range manager.Clients() {
			if client.State().Status == mcpcore.ClientConnected {
				return manager
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("mcp prompt manager did not connect")

	return nil
}
