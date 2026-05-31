package strategy

import (
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
)

type Strategy interface {
	Resolve(permissions model.Permissions, call kit.ToolCall) model.ResolveResult
}

type defaultStrategy struct{}

func Resolve(permissions model.Permissions, call kit.ToolCall) model.ResolveResult {
	return forTool(call.Name).Resolve(permissions, call)
}

func forTool(tool string) Strategy {
	if strings.HasPrefix(tool, mcpToolPrefix) {
		return mcpStrategy{}
	}

	switch tool {
	case ToolReadFile, ToolWriteFile, ToolEditFile:
		return pathStrategy{}
	case ToolList:
		return pathStrategy{directory: true}
	case ToolWebFetch:
		return urlStrategy{}
	case ToolBash:
		return bashStrategy{}
	default:
		return defaultStrategy{}
	}
}

func (defaultStrategy) Resolve(permissions model.Permissions, call kit.ToolCall) model.ResolveResult {
	input := extractInput(call)

	if matchesRuleSet(permissions.Deny, call.Name, input, nil) {
		return resolveResult(model.Deny, call, input, nil)
	}

	if matchesRuleSet(permissions.Ask, call.Name, input, nil) {
		return resolveResult(model.Ask, call, input, nil)
	}

	if matchesRuleSet(permissions.Allow, call.Name, input, nil) {
		return resolveResult(model.Allow, call, input, nil)
	}

	return resolveResult(permissions.Default, call, input, nil)
}

func matchesRuleSet(rules []model.Rule, tool, input string, match func(pattern, input string) bool) bool {
	for _, rule := range rules {
		if ruleMatches(rule, tool, input, match) {
			return true
		}
	}

	return false
}

func ruleMatches(rule model.Rule, tool, input string, match func(pattern, input string) bool) bool {
	if rule.Tool != tool {
		return false
	}

	if rule.Pattern == "" || rule.Pattern == "*" {
		return true
	}

	if match == nil {
		return false
	}

	return match(rule.Pattern, input)
}

func resolveResult(decision model.Decision, call kit.ToolCall, input string, options []model.AskOption) model.ResolveResult {
	if decision != model.Ask {
		return model.ResolveResult{Decision: decision}
	}

	request := model.NewAskRequest(call, input, options)

	return model.ResolveResult{
		Decision:   decision,
		AskRequest: &request,
	}
}
