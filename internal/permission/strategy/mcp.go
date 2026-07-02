package strategy

import (
	"fmt"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/utils/glob"
)

const mcpToolPrefix = "mcp__"

type mcpStrategy struct{}

func (mcpStrategy) Resolve(permissions model.Permissions, call kit.ToolCall) model.ResolveResult {
	if matchesMCPRuleSet(permissions.Deny, call.Name) {
		return resolveResult(model.Deny, call, "", nil)
	}

	options := mcpOptions(call.Name)
	if matchesMCPRuleSet(permissions.Ask, call.Name) {
		return resolveResult(model.Ask, call, "", options)
	}

	if matchesMCPRuleSet(permissions.Allow, call.Name) {
		return resolveResult(model.Allow, call, "", nil)
	}

	return resolveResult(permissions.Default, call, "", options)
}

func matchesMCPRuleSet(rules []model.Rule, tool string) bool {
	for _, rule := range rules {
		if matchToolName(rule.Tool, tool) {
			return true
		}
	}

	return false
}

func matchToolName(pattern, tool string) bool {
	if pattern == "" {
		return false
	}

	if pattern == "*" {
		return true
	}

	return glob.Match(pattern, tool)
}

func mcpOptions(tool string) []model.Option {
	options := []model.Option{
		{
			ID:       model.OptionAllowExact,
			Label:    "Allow this tool",
			Decision: model.Allow,
			Scope:    model.ScopeGlobal,
			Rule:     &model.Rule{Tool: tool},
		},
	}

	if server, ok := mcpServer(tool); ok {
		options = append(options, model.Option{
			ID:       model.OptionAllowPattern,
			Label:    fmt.Sprintf("Allow all tools from %s", server),
			Decision: model.Allow,
			Scope:    model.ScopeGlobal,
			Rule:     &model.Rule{Tool: mcpToolPrefix + server + "__*"},
		})
	}

	return options
}

func mcpServer(tool string) (string, bool) {
	rest, ok := strings.CutPrefix(tool, mcpToolPrefix)
	if !ok {
		return "", false
	}

	server, _, ok := strings.Cut(rest, "__")
	if !ok || server == "" {
		return "", false
	}

	return server, true
}
