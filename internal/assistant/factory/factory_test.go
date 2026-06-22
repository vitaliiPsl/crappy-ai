package factory

import (
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/tool"
)

func toolSpec(name string) ToolSpec {
	t := tool.MustNew(name, "test tool", func(_ *kit.RunContext, _ struct{}) (string, error) {
		return "", nil
	})

	return ToolSpec{Source: "test", Tool: t}
}

func names(tools []ToolSpec) []string {
	out := make([]string, len(tools))
	for i, t := range tools {
		out[i] = t.Name()
	}

	return out
}

func TestAllowedToolsEmptyAllowlistKeepsAll(t *testing.T) {
	tools := []ToolSpec{toolSpec("read_file"), toolSpec("bash")}

	got := allowedTools(tools, nil)
	if len(got) != 2 {
		t.Fatalf("tools = %v, want all kept when allowlist empty", names(got))
	}
}

func TestAllowedToolsFiltersToAllowlist(t *testing.T) {
	tools := []ToolSpec{toolSpec("read_file"), toolSpec("bash"), toolSpec("list")}

	got := allowedTools(tools, []string{"read_file", "list"})

	want := map[string]bool{"read_file": true, "list": true}
	if len(got) != len(want) {
		t.Fatalf("tools = %v, want %v", names(got), want)
	}

	for _, t2 := range got {
		if !want[t2.Name()] {
			t.Fatalf("unexpected tool %q in filtered set", t2.Name())
		}
	}
}
