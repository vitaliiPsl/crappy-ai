package permission

import (
	"net/url"
	"strings"

	"github.com/vitaliiPsl/crappy-ai/internal/utils"
	"github.com/vitaliiPsl/crappy-ai/internal/utils/glob"
)

func matches(rule Rule, tool, input string) bool {
	if rule.Tool != tool {
		return false
	}

	if rule.Pattern == "" || rule.Pattern == "*" {
		return true
	}

	switch tool {
	case "web_fetch":
		return matchURL(rule.Pattern, input)
	case "bash":
		return matchBash(rule.Pattern, input)
	case "read_file", "write_file", "edit_file", "list":
		return matchPath(rule.Pattern, input)
	default:
		return false
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

func matchURL(pattern, rawURL string) bool {
	domainPattern, ok := strings.CutPrefix(pattern, "domain:")
	if !ok {
		return false
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return false
	}

	return matchDomain(domainPattern, u.Hostname())
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
