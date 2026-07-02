package strategy

import (
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
)

func mcpCall(tool string) kit.ToolCall {
	return kit.NewToolCall("call_1", tool, map[string]any{"q": "x"})
}

func TestResolveMCPAllowsExactTool(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Allow:   []model.Rule{{Tool: "mcp__github__search"}},
	}

	if got := Resolve(perms, mcpCall("mcp__github__search")).Decision; got != model.Allow {
		t.Fatalf("Resolve = %q, want %q", got, model.Allow)
	}
}

func TestResolveMCPAllowsWholeServer(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Allow:   []model.Rule{{Tool: "mcp__github__*"}},
	}

	if got := Resolve(perms, mcpCall("mcp__github__search")).Decision; got != model.Allow {
		t.Fatalf("Resolve = %q, want %q", got, model.Allow)
	}
}

func TestResolveMCPServerWildcardDoesNotLeakAcrossServers(t *testing.T) {
	perms := model.Permissions{
		Default: model.Deny,
		Allow:   []model.Rule{{Tool: "mcp__github__*"}},
	}

	if got := Resolve(perms, mcpCall("mcp__gitlab__search")).Decision; got != model.Deny {
		t.Fatalf("Resolve = %q, want %q", got, model.Deny)
	}
}

func TestResolveMCPDenyTakesPrecedence(t *testing.T) {
	perms := model.Permissions{
		Default: model.Allow,
		Deny:    []model.Rule{{Tool: "mcp__github__*"}},
		Allow:   []model.Rule{{Tool: "mcp__github__search"}},
	}

	if got := Resolve(perms, mcpCall("mcp__github__search")).Decision; got != model.Deny {
		t.Fatalf("Resolve = %q, want %q", got, model.Deny)
	}
}

func TestResolveMCPMatchesAllMCPTools(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Allow:   []model.Rule{{Tool: "mcp__*"}},
	}

	if got := Resolve(perms, mcpCall("mcp__deepwiki__ask_question")).Decision; got != model.Allow {
		t.Fatalf("Resolve = %q, want %q", got, model.Allow)
	}
}

func TestResolveMCPAskOffersToolAndServerOptions(t *testing.T) {
	request := prompt(t, mcpCall("mcp__github__search"))

	if rule := optionRule(t, request, model.OptionAllowExact); rule.Tool != "mcp__github__search" {
		t.Fatalf("allow-exact rule tool = %q, want mcp__github__search", rule.Tool)
	}

	if rule := optionRule(t, request, model.OptionAllowPattern); rule.Tool != "mcp__github__*" {
		t.Fatalf("allow-pattern rule tool = %q, want mcp__github__*", rule.Tool)
	}
}
