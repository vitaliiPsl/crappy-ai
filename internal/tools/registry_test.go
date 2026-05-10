package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type fakeTool struct {
	name string
}

func (t fakeTool) Definition() kit.ToolDefinition {
	return kit.ToolDefinition{Name: t.name}
}

func (t fakeTool) Execute(_ context.Context, _ map[string]any) (string, error) {
	return "", nil
}

func newTestRegistry(t *testing.T, tools ...kit.Tool) *Registry {
	t.Helper()

	entries := make(map[string]kit.Tool)
	registerTools(entries, tools...)

	return &Registry{entries: entries}
}

func TestGetTools(t *testing.T) {
	registry := NewRegistry()

	want := []string{
		"bash",
		"edit_file",
		"list",
		"read_file",
		"web_fetch",
		"write_file",
	}

	got := toolNames(registry.GetTools())
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("tool names = %v, want %v", got, want)
	}
}

func TestGetTool(t *testing.T) {
	registry := newTestRegistry(t, fakeTool{name: "alpha"})

	tool, err := registry.GetTool("alpha")
	if err != nil {
		t.Fatalf("GetTool: %v", err)
	}

	if tool.Definition().Name != "alpha" {
		t.Fatalf("tool name = %q, want alpha", tool.Definition().Name)
	}
}

func TestGetTool_UnknownIncludesAvailableTools(t *testing.T) {
	registry := newTestRegistry(t, fakeTool{name: "beta"}, fakeTool{name: "alpha"})

	_, err := registry.GetTool("missing")
	if err == nil {
		t.Fatal("GetTool missing: want error")
	}

	msg := err.Error()
	if !strings.Contains(msg, `unknown tool "missing"`) {
		t.Fatalf("error = %q, want unknown tool", msg)
	}

	if !strings.Contains(msg, "[alpha beta]") {
		t.Fatalf("error = %q, want sorted available tools", msg)
	}
}

func TestRegisterTools_DuplicatePanics(t *testing.T) {
	entries := make(map[string]kit.Tool)
	registerTools(entries, fakeTool{name: "alpha"})

	defer func() {
		if recover() == nil {
			t.Fatal("registerTools duplicate did not panic")
		}
	}()

	registerTools(entries, fakeTool{name: "alpha"})
}

func toolNames(tools []kit.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Definition().Name
	}

	return names
}
