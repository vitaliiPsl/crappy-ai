package mcp

import (
	"context"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestToolExecuteCallsClientTool(t *testing.T) {
	client := &fakeClient{
		config: Config{Name: "github"},
		result: kit.NewToolResult(kit.NewToolCall("", "search", nil), "done", nil),
	}

	output, err := newTool("github", client, kit.ToolDefinition{Name: "search"}).Execute(
		kit.NewRunContext(context.Background()),
		map[string]any{"q": "x"},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if output != "done" {
		t.Fatalf("output = %q, want done", output)
	}

	if client.called.Name != "search" {
		t.Fatalf("called name = %q, want search", client.called.Name)
	}

	if client.called.Arguments["q"] != "x" {
		t.Fatalf("called arguments = %#v, want q=x", client.called.Arguments)
	}
}
