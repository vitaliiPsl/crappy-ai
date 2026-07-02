package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
)

func TestCoreContributorTools(t *testing.T) {
	got, err := coreContributor{}.Contribute(context.Background(), appagent.Request{})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	want := []string{
		"bash",
		"web_fetch",
		"read_file",
		"write_file",
		"edit_file",
		"list",
	}

	if names := toolNames(got.Tools); strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("tool names = %v, want %v", names, want)
	}
}

func toolNames(tools []kit.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Definition().Name
	}

	return names
}
