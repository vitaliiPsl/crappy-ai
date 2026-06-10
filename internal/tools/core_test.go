package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/background"
)

func TestCore(t *testing.T) {
	manager := background.NewManager(context.Background())
	defer manager.Close()

	want := []string{
		"bash",
		"web_fetch",
		"read_file",
		"write_file",
		"edit_file",
		"list",
	}

	got := toolNames(Core(manager))
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("tool names = %v, want %v", got, want)
	}
}

func toolNames(tools []kit.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Definition().Name
	}

	return names
}
