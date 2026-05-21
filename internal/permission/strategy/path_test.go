package strategy

import (
	"path/filepath"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
)

func TestResolvePathOrder(t *testing.T) {
	perms := model.Permissions{
		Default: model.Allow,
		Deny:    []model.Rule{{Tool: ToolReadFile, Pattern: "//etc/**"}},
		Ask:     []model.Rule{{Tool: ToolReadFile, Pattern: "//etc/passwd"}},
		Allow:   []model.Rule{{Tool: ToolReadFile, Pattern: "**"}},
	}

	got := Resolve(perms, kit.NewToolCall("call_1", ToolReadFile, map[string]any{
		inputPath: "/etc/passwd",
	})).Decision

	if got != model.Deny {
		t.Fatalf("Resolve = %q, want %q", got, model.Deny)
	}
}

func TestResolvePathAllowsMatchingRule(t *testing.T) {
	root := t.TempDir()
	call := kit.NewToolCall("call_1", ToolReadFile, map[string]any{
		inputPath: filepath.Join(root, "docs", "guide.md"),
	})
	perms := model.Permissions{
		Default: model.Ask,
		Allow: []model.Rule{
			{Tool: ToolReadFile, Pattern: recursivePathPattern(filepath.Join(root, "docs"))},
		},
	}

	got := Resolve(perms, call)
	if got.Decision != model.Allow {
		t.Fatalf("Resolve decision = %q, want %q", got.Decision, model.Allow)
	}

	if got.AskRequest != nil {
		t.Fatalf("AskRequest = %+v, want nil", got.AskRequest)
	}
}

func TestResolvePathAskRuleIncludesOptions(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "guide.md")
	call := kit.NewToolCall("call_1", ToolReadFile, map[string]any{inputPath: path})
	perms := model.Permissions{
		Default: model.Allow,
		Ask: []model.Rule{
			{Tool: ToolReadFile, Pattern: recursivePathPattern(filepath.Join(root, "docs"))},
		},
	}

	got := Resolve(perms, call)
	if got.Decision != model.Ask {
		t.Fatalf("Resolve decision = %q, want %q", got.Decision, model.Ask)
	}

	if got.AskRequest == nil {
		t.Fatal("AskRequest = nil, want ask request")
	}

	if got.AskRequest.Input != path {
		t.Fatalf("AskRequest input = %q, want %q", got.AskRequest.Input, path)
	}

	exact := optionRule(t, *got.AskRequest, model.OptionAllowExact)
	if exact != (model.Rule{Tool: ToolReadFile, Pattern: permissionPath(path)}) {
		t.Fatalf("exact rule = %+v, want exact read_file path", exact)
	}
}

func TestPathAskRequestFileOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docs", "guide.md")
	request := askRequest(t, kit.NewToolCall("call_1", ToolReadFile, map[string]any{inputPath: path}))

	exact := optionRule(t, request, model.OptionAllowExact)
	if exact != (model.Rule{Tool: ToolReadFile, Pattern: permissionPath(path)}) {
		t.Fatalf("exact rule = %+v, want exact read_file path", exact)
	}

	pattern := optionRule(t, request, model.OptionAllowPattern)

	wantPattern := recursivePathPattern(filepath.Dir(path))
	if pattern != (model.Rule{Tool: ToolReadFile, Pattern: wantPattern}) {
		t.Fatalf("pattern rule = %+v, want read_file %q", pattern, wantPattern)
	}
}

func TestPathAskRequestListPatternUsesListedDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docs")
	request := askRequest(t, kit.NewToolCall("call_1", ToolList, map[string]any{inputPath: path}))

	exact := optionRule(t, request, model.OptionAllowExact)
	if exact != (model.Rule{Tool: ToolList, Pattern: permissionPath(path)}) {
		t.Fatalf("exact rule = %+v, want exact list path", exact)
	}

	pattern := optionRule(t, request, model.OptionAllowPattern)

	wantPattern := recursivePathPattern(path)
	if pattern != (model.Rule{Tool: ToolList, Pattern: wantPattern}) {
		t.Fatalf("pattern rule = %+v, want list %q", pattern, wantPattern)
	}
}

func TestMatchPathConvertsRelativeInputsToAbsolute(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{
			name:    "relative input matches absolute pattern",
			pattern: filepath.Join(root, "internal", "**"),
			input:   "internal/permission/service.go",
			want:    true,
		},
		{
			name:    "relative pattern matches absolute input",
			pattern: "internal/**",
			input:   filepath.Join(root, "internal", "permission", "service.go"),
			want:    true,
		},
		{
			name:    "dot dot is cleaned before matching",
			pattern: filepath.Join(root, "internal", "**"),
			input:   "internal/../outside.txt",
			want:    false,
		},
		{
			name:    "rooted pattern still matches absolute input",
			pattern: "//" + filepath.ToSlash(filepath.Join(root, "internal")) + "/**",
			input:   filepath.Join(root, "internal", "permission", "service.go"),
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchPath(tt.pattern, tt.input); got != tt.want {
				t.Fatalf("matchPath(%q, %q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchPathGlobSemantics(t *testing.T) {
	root := t.TempDir()

	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{
			name:    "star matches within one segment",
			pattern: filepath.Join(root, "docs", "*.md"),
			input:   filepath.Join(root, "docs", "guide.md"),
			want:    true,
		},
		{
			name:    "star does not cross directories",
			pattern: filepath.Join(root, "docs", "*.md"),
			input:   filepath.Join(root, "docs", "nested", "guide.md"),
			want:    false,
		},
		{
			name:    "doublestar matches nested directories",
			pattern: filepath.Join(root, "docs", "**"),
			input:   filepath.Join(root, "docs", "nested", "guide.md"),
			want:    true,
		},
		{
			name:    "doublestar matches zero segments",
			pattern: filepath.Join(root, "docs", "**"),
			input:   filepath.Join(root, "docs"),
			want:    true,
		},
		{
			name:    "question matches one character",
			pattern: filepath.Join(root, "docs", "guide-?.md"),
			input:   filepath.Join(root, "docs", "guide-1.md"),
			want:    true,
		},
		{
			name:    "character class matches one character",
			pattern: filepath.Join(root, "docs", "guide-[ab].md"),
			input:   filepath.Join(root, "docs", "guide-a.md"),
			want:    true,
		},
		{
			name:    "different absolute path does not match",
			pattern: filepath.Join(root, "docs", "**"),
			input:   filepath.Join(root, "other", "guide.md"),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchPath(tt.pattern, tt.input); got != tt.want {
				t.Fatalf("matchPath(%q, %q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestRecursivePathPatternRoot(t *testing.T) {
	if got := recursivePathPattern(string(filepath.Separator)); got != "/**" {
		t.Fatalf("recursivePathPattern(root) = %q, want /**", got)
	}
}
