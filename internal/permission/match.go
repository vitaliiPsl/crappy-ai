package permission

import (
	"path/filepath"
	"regexp"
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
	case "read_file", "write_file", "edit_file", "list":
		return matchPath(rule.Pattern, input)
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

func matchPath(pattern, input string) bool {
	return glob.Match(absPath(pattern), absPath(input))
}

func absPath(path string) string {
	path = utils.ExpandHome(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}

	return abs
}
