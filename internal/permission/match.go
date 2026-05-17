package permission

import (
	"regexp"
	"strings"

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
	case "read_file", "write_file", "edit_file", "list":
		return glob.Match(rule.Pattern, input)
	default:
		return false
	}
}

func matchURL(pattern, rawURL string) bool {
	escaped := regexp.QuoteMeta(pattern)
	regex := strings.ReplaceAll(escaped, `\*`, `.*`)

	re, err := regexp.Compile("^" + regex + "$")
	if err != nil {
		return false
	}

	return re.MatchString(rawURL)
}
