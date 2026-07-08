package mcp

import (
	"context"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestToolExecuteCallsClientTool(t *testing.T) {
	client := &fakeClient{
		config: Config{Name: "github"},
		result: kit.NewToolResult(kit.NewToolCall("", "search", nil), kit.NewToolOutput(kit.NewTextContent("done")), nil),
	}

	output, err := newTool("github", client, kit.ToolDefinition{Name: "search"}).Execute(
		kit.NewRunContext(context.Background()),
		kit.NewToolCall("call-1", "mcp__github__search", map[string]any{"q": "x"}),
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := kit.ContentsText(output.Content); got != "done" {
		t.Fatalf("output = %q, want done", got)
	}

	if client.called.Name != "search" {
		t.Fatalf("called name = %q, want search", client.called.Name)
	}

	if client.called.Arguments["q"] != "x" {
		t.Fatalf("called arguments = %#v, want q=x", client.called.Arguments)
	}
}
