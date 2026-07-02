package strategy

import (
	"net/url"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/utils/glob"
)

type urlStrategy struct{}

func (urlStrategy) Resolve(permissions model.Permissions, call kit.ToolCall) model.ResolveResult {
	input := extractInput(call)

	if matchesRuleSet(permissions.Deny, call.Name, input, matchURL) {
		return resolveResult(model.Deny, call, input, nil)
	}

	options := urlOptions(call.Name, input)
	if matchesRuleSet(permissions.Ask, call.Name, input, matchURL) {
		return resolveResult(model.Ask, call, input, options)
	}

	if matchesRuleSet(permissions.Allow, call.Name, input, matchURL) {
		return resolveResult(model.Allow, call, input, nil)
	}

	return resolveResult(permissions.Default, call, input, options)
}

func urlOptions(tool, input string) []model.Option {
	u, err := url.Parse(input)
	if err != nil || u.Host == "" {
		return nil
	}

	host := normalizeDomain(u.Hostname())
	if host == "" {
		return nil
	}

	input = strings.TrimSpace(input)

	return []model.Option{
		{
			ID:       model.OptionAllowExact,
			Label:    "Allow exact URL",
			Decision: model.Allow,
			Scope:    model.ScopeGlobal,
			Rule:     &model.Rule{Tool: tool, Pattern: "url:" + input},
		},
		{
			ID:       model.OptionAllowPattern,
			Label:    "Allow domain",
			Decision: model.Allow,
			Scope:    model.ScopeGlobal,
			Rule:     &model.Rule{Tool: tool, Pattern: "domain:" + host},
		},
	}
}

func matchURL(pattern, rawURL string) bool {
	if exactURL, ok := strings.CutPrefix(pattern, "url:"); ok {
		return strings.TrimSpace(exactURL) == strings.TrimSpace(rawURL)
	}

	if domainPattern, ok := strings.CutPrefix(pattern, "domain:"); ok {
		u, err := url.Parse(rawURL)
		if err != nil || u.Host == "" {
			return false
		}

		return matchDomain(domainPattern, u.Hostname())
	}

	return false
}

func matchDomain(pattern, host string) bool {
	pattern = normalizeDomain(pattern)

	host = normalizeDomain(host)
	if pattern == "" || host == "" {
		return false
	}

	if pattern == "*" {
		return true
	}

	return glob.MatchSegments(strings.Split(pattern, "."), strings.Split(host, "."))
}

func normalizeDomain(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".")

	return strings.ToLower(s)
}
