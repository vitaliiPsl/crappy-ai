package strategy

import (
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
)

func webFetchCall(rawURL string) kit.ToolCall {
	return kit.NewToolCall("call_1", ToolWebFetch, map[string]any{inputURL: rawURL})
}

func TestResolveURLOrder(t *testing.T) {
	perms := model.Permissions{
		Default: model.Allow,
		Deny:    []model.Rule{{Tool: ToolWebFetch, Pattern: "domain:example.com"}},
		Ask:     []model.Rule{{Tool: ToolWebFetch, Pattern: "url:https://example.com/docs"}},
		Allow:   []model.Rule{{Tool: ToolWebFetch, Pattern: "domain:*"}},
	}

	got := Resolve(perms, webFetchCall("https://example.com/docs")).Decision
	if got != model.Deny {
		t.Fatalf("Resolve = %q, want %q", got, model.Deny)
	}
}

func TestResolveURLAllowsMatchingRule(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Allow: []model.Rule{
			{Tool: ToolWebFetch, Pattern: "domain:docs.example.com"},
		},
	}

	got := Resolve(perms, webFetchCall("https://docs.example.com/guide"))
	if got.Decision != model.Allow {
		t.Fatalf("Resolve decision = %q, want %q", got.Decision, model.Allow)
	}

	if got.Prompt != nil {
		t.Fatalf("Prompt = %+v, want nil", got.Prompt)
	}
}

func TestResolveURLAskRuleIncludesOptions(t *testing.T) {
	rawURL := "https://docs.example.com/guide?x=1"
	perms := model.Permissions{
		Default: model.Allow,
		Ask: []model.Rule{
			{Tool: ToolWebFetch, Pattern: "domain:docs.example.com"},
		},
	}

	got := Resolve(perms, webFetchCall(rawURL))
	if got.Decision != model.Ask {
		t.Fatalf("Resolve decision = %q, want %q", got.Decision, model.Ask)
	}

	if got.Prompt == nil {
		t.Fatal("Prompt = nil, want prompt")
	}

	if got.Prompt.Input != rawURL {
		t.Fatalf("Prompt input = %q, want %q", got.Prompt.Input, rawURL)
	}

	exact := optionRule(t, *got.Prompt, model.OptionAllowExact)
	if exact != (model.Rule{Tool: ToolWebFetch, Pattern: "url:" + rawURL}) {
		t.Fatalf("exact rule = %+v, want exact web_fetch URL", exact)
	}
}

func TestURLPromptOptions(t *testing.T) {
	rawURL := "https://docs.example.com/guide?x=1"
	request := prompt(t, webFetchCall(rawURL))

	exact := optionRule(t, request, model.OptionAllowExact)
	if exact != (model.Rule{Tool: ToolWebFetch, Pattern: "url:" + rawURL}) {
		t.Fatalf("exact rule = %+v, want exact web_fetch URL", exact)
	}

	pattern := optionRule(t, request, model.OptionAllowPattern)
	if pattern != (model.Rule{Tool: ToolWebFetch, Pattern: "domain:docs.example.com"}) {
		t.Fatalf("pattern rule = %+v, want domain rule", pattern)
	}
}

func TestURLPromptOptionsNormalizeDomain(t *testing.T) {
	request := prompt(t, webFetchCall("https://DOCS.Example.COM:443/guide"))

	pattern := optionRule(t, request, model.OptionAllowPattern)
	if pattern != (model.Rule{Tool: ToolWebFetch, Pattern: "domain:docs.example.com"}) {
		t.Fatalf("pattern rule = %+v, want normalized domain rule", pattern)
	}
}

func TestURLPromptInvalidURLHasOnlyOnceOptions(t *testing.T) {
	request := prompt(t, webFetchCall("not a url"))

	if len(request.Options) != 2 {
		t.Fatalf("options = %#v, want allow once and deny only", request.Options)
	}

	if _, ok := request.Option(model.OptionAllowOnce); !ok {
		t.Fatalf("allow once option missing from %#v", request.Options)
	}

	if _, ok := request.Option(model.OptionDenyOnce); !ok {
		t.Fatalf("deny option missing from %#v", request.Options)
	}
}

func TestMatchURLExact(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		url     string
		want    bool
	}{
		{
			name:    "exact url matches",
			pattern: "url:https://example.com/docs",
			url:     "https://example.com/docs",
			want:    true,
		},
		{
			name:    "different path does not match",
			pattern: "url:https://example.com/docs",
			url:     "https://example.com/other",
			want:    false,
		},
		{
			name:    "surrounding whitespace ignored",
			pattern: "url: https://example.com/docs ",
			url:     " https://example.com/docs ",
			want:    true,
		},
		{
			name:    "url pattern does not canonicalize case",
			pattern: "url:https://EXAMPLE.com/docs",
			url:     "https://example.com/docs",
			want:    false,
		},
		{
			name:    "bare glob-like url is not a supported pattern",
			pattern: "https://example.com/*",
			url:     "https://example.com/docs",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchURL(tt.pattern, tt.url); got != tt.want {
				t.Fatalf("matchURL(%q, %q) = %v, want %v", tt.pattern, tt.url, got, tt.want)
			}
		})
	}
}

func TestMatchURLDomain(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		url     string
		want    bool
	}{
		{
			name:    "domain matches root",
			pattern: "domain:example.com",
			url:     "https://example.com/",
			want:    true,
		},
		{
			name:    "domain matches any path",
			pattern: "domain:example.com",
			url:     "https://example.com/docs/intro",
			want:    true,
		},
		{
			name:    "different domain does not match",
			pattern: "domain:example.com",
			url:     "https://other.com/",
			want:    false,
		},
		{
			name:    "domain is case insensitive",
			pattern: "domain:example.com",
			url:     "https://EXAMPLE.com/",
			want:    true,
		},
		{
			name:    "port is ignored",
			pattern: "domain:example.com",
			url:     "https://example.com:443/docs",
			want:    true,
		},
		{
			name:    "single star matches one label",
			pattern: "domain:*.example.com",
			url:     "https://api.example.com/v1",
			want:    true,
		},
		{
			name:    "single star does not match zero labels",
			pattern: "domain:*.example.com",
			url:     "https://example.com/v1",
			want:    false,
		},
		{
			name:    "single star does not match multiple labels",
			pattern: "domain:*.example.com",
			url:     "https://v1.api.example.com/",
			want:    false,
		},
		{
			name:    "doublestar matches multiple labels",
			pattern: "domain:**.example.com",
			url:     "https://v1.api.example.com/",
			want:    true,
		},
		{
			name:    "star can appear inside label",
			pattern: "domain:api-*.example.com",
			url:     "https://api-v1.example.com/",
			want:    true,
		},
		{
			name:    "question can appear inside label",
			pattern: "domain:ex?mple.com",
			url:     "https://example.com/",
			want:    true,
		},
		{
			name:    "star can match tld label",
			pattern: "domain:example.*",
			url:     "https://example.org/",
			want:    true,
		},
		{
			name:    "host-like path does not affect matching",
			pattern: "domain:*.example.com",
			url:     "https://evil.com/api.example.com/",
			want:    false,
		},
		{
			name:    "global wildcard matches host",
			pattern: "domain:*",
			url:     "https://example.com/",
			want:    true,
		},
		{
			name:    "invalid url does not match domain",
			pattern: "domain:example.com",
			url:     "not a url",
			want:    false,
		},
		{
			name:    "empty domain pattern does not match",
			pattern: "domain:",
			url:     "https://example.com/",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchURL(tt.pattern, tt.url); got != tt.want {
				t.Fatalf("matchURL(%q, %q) = %v, want %v", tt.pattern, tt.url, got, tt.want)
			}
		})
	}
}

func TestNormalizeDomain(t *testing.T) {
	if got := normalizeDomain(" Example.COM. "); got != "example.com" {
		t.Fatalf("normalizeDomain = %q, want example.com", got)
	}
}
