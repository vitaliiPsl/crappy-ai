package strategy

import (
	"path/filepath"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/utils"
	"github.com/vitaliiPsl/crappy-ai/internal/utils/glob"
)

type pathStrategy struct {
	directory bool
}

func (s pathStrategy) Resolve(permissions model.Permissions, call kit.ToolCall) model.ResolveResult {
	input := extractInput(call)

	if matchesRuleSet(permissions.Deny, call.Name, input, matchPath) {
		return resolveResult(model.Deny, call, input, nil)
	}

	options := s.options(call.Name, input)
	if matchesRuleSet(permissions.Ask, call.Name, input, matchPath) {
		return resolveResult(model.Ask, call, input, options)
	}

	if matchesRuleSet(permissions.Allow, call.Name, input, matchPath) {
		return resolveResult(model.Allow, call, input, nil)
	}

	return resolveResult(permissions.Default, call, input, options)
}

func (s pathStrategy) options(tool, input string) []model.Option {
	abs, err := utils.AbsPath(input)
	if err != nil || abs == "" {
		return nil
	}

	patternBase := abs
	if !s.directory {
		patternBase = filepath.Dir(abs)
	}

	return []model.Option{
		{
			ID:       model.OptionAllowExact,
			Label:    "Allow exact path",
			Decision: model.Allow,
			Scope:    model.ScopeGlobal,
			Rule:     &model.Rule{Tool: tool, Pattern: permissionPath(abs)},
		},
		{
			ID:       model.OptionAllowPattern,
			Label:    "Allow path pattern",
			Decision: model.Allow,
			Scope:    model.ScopeGlobal,
			Rule:     &model.Rule{Tool: tool, Pattern: recursivePathPattern(patternBase)},
		},
	}
}

func matchPath(pattern, input string) bool {
	pattern, err := utils.AbsPath(pattern)
	if err != nil {
		return false
	}

	input, err = utils.AbsPath(input)
	if err != nil {
		return false
	}

	return glob.Match(pattern, input)
}

func permissionPath(path string) string {
	return "/" + filepath.ToSlash(path)
}

func recursivePathPattern(path string) string {
	if path == string(filepath.Separator) {
		return "/**"
	}

	return permissionPath(path) + "/**"
}
