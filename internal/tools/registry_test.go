package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/kittest"

	"github.com/vitaliiPsl/crappy-ai/internal/background"
)

func newTestRegistry(t *testing.T, tools ...kit.Tool) *Registry {
	t.Helper()

	entries := make(map[string]kit.Tool)
	registerTools(entries, tools...)

	return &Registry{entries: entries}
}

func TestGetTools(t *testing.T) {
	manager := background.NewManager(context.Background())
	defer manager.Close()

	registry := NewRegistry(manager)

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
	registry := newTestRegistry(t, kittest.NewTool(t, "alpha", ""))

	tool, err := registry.GetTool("alpha")
	if err != nil {
		t.Fatalf("GetTool: %v", err)
	}

	if tool.Definition().Name != "alpha" {
		t.Fatalf("tool name = %q, want alpha", tool.Definition().Name)
	}
}

func TestGetTool_UnknownIncludesAvailableTools(t *testing.T) {
	registry := newTestRegistry(t, kittest.NewTool(t, "beta", ""), kittest.NewTool(t, "alpha", ""))

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
	registerTools(entries, kittest.NewTool(t, "alpha", ""))

	defer func() {
		if recover() == nil {
			t.Fatal("registerTools duplicate did not panic")
		}
	}()

	registerTools(entries, kittest.NewTool(t, "alpha", ""))
}

func toolNames(tools []kit.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Definition().Name
	}

	return names
}
