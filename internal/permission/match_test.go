package permission

import (
	"path/filepath"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestResolveOrder(t *testing.T) {
	perms := Permissions{
		Default: Allow,
		Deny:    []Rule{{Tool: "read_file", Pattern: "//etc/**"}},
		Allow:   []Rule{{Tool: "read_file", Pattern: "**"}},
	}

	got := Resolve(perms, kit.NewToolCall("call_1", "read_file", map[string]any{
		"path": "/etc/passwd",
	}))

	if got != Deny {
		t.Fatalf("Resolve = %q, want %q", got, Deny)
	}
}

func TestMatchURL(t *testing.T) {
	tests := []struct {
		pattern string
		url     string
		want    bool
	}{
		{pattern: "https://example.com/*", url: "https://example.com/", want: true},
		{pattern: "https://example.com/*", url: "https://example.com/docs/intro", want: true},
		{pattern: "https://example.com/*", url: "https://other.com/", want: false},
		{pattern: "https://*.example.com/*", url: "https://api.example.com/v1", want: true},
		{pattern: "https://*.example.com/*", url: "https://example.com/v1", want: false},
		{pattern: "https://example.com/docs/*", url: "https://example.com/blog/post", want: false},
		{pattern: "https://example.com", url: "https://example.com", want: true},
		{pattern: "https://example.com", url: "https://example.com/", want: false},
	}

	for _, tt := range tests {
		got := matchURL(tt.pattern, tt.url)
		if got != tt.want {
			t.Fatalf("matchURL(%q, %q) = %v, want %v", tt.pattern, tt.url, got, tt.want)
		}
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
