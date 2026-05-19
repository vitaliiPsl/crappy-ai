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
		{pattern: "domain:example.com", url: "https://example.com/", want: true},
		{pattern: "domain:example.com", url: "https://example.com/docs/intro", want: true},
		{pattern: "domain:example.com", url: "https://other.com/", want: false},
		{pattern: "domain:example.com", url: "https://EXAMPLE.com/", want: true},
		{pattern: "domain:example.com", url: "https://example.com:443/docs", want: true},
		{pattern: "domain:*.example.com", url: "https://api.example.com/v1", want: true},
		{pattern: "domain:*.example.com", url: "https://example.com/v1", want: false},
		{pattern: "domain:*.example.com", url: "https://v1.api.example.com/", want: false},
		{pattern: "domain:**.example.com", url: "https://v1.api.example.com/", want: true},
		{pattern: "domain:example.*", url: "https://example.org/", want: true},
		{pattern: "domain:api-*.example.com", url: "https://api-v1.example.com/", want: true},
		{pattern: "domain:ex?mple.com", url: "https://example.com/", want: true},
		{pattern: "domain:*.example.com", url: "https://evil.com/api.example.com/", want: false},
		{pattern: "https://example.com/*", url: "https://example.com/docs", want: false},
		{pattern: "domain:example.com", url: "not a url", want: false},
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
