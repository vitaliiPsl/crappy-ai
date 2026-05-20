package permission

import (
	"regexp"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

func resolveBash(permissions Permissions, command string) Decision {
	command = strings.TrimSpace(command)
	parts, hasSubstitution := analyzeBashCommand(command)

	for _, rule := range permissions.Deny {
		if bashRuleMatchesAny(rule, command, parts) {
			return Deny
		}
	}

	for _, rule := range permissions.Ask {
		if bashRuleMatchesAny(rule, command, parts) {
			return Ask
		}
	}

	if bashAllowed(permissions.Allow, command, parts, hasSubstitution) {
		return Allow
	}

	return permissions.Default
}

func bashAllowed(rules []Rule, command string, parts []string, hasSubstitution bool) bool {
	for _, rule := range rules {
		if rule.Tool != "bash" {
			continue
		}

		if rule.Pattern == "" || rule.Pattern == "*" {
			return true
		}

		if !hasCommandWildcard(rule.Pattern) && matchCommandPattern(rule.Pattern, command) {
			return true
		}
	}

	if hasSubstitution || len(parts) == 0 {
		return false
	}

	for _, part := range parts {
		if !bashPartAllowed(rules, part) {
			return false
		}
	}

	return true
}

func bashPartAllowed(rules []Rule, command string) bool {
	for _, rule := range rules {
		if rule.Tool == "bash" && matchBash(rule.Pattern, command) {
			return true
		}
	}

	return false
}

func bashRuleMatchesAny(rule Rule, command string, parts []string) bool {
	if rule.Tool != "bash" {
		return false
	}

	if matchBash(rule.Pattern, command) {
		return true
	}

	for _, part := range parts {
		if matchBash(rule.Pattern, part) {
			return true
		}
	}

	return false
}

func matchBash(pattern, command string) bool {
	pattern = strings.TrimSpace(pattern)
	command = strings.TrimSpace(command)

	if pattern == "" || pattern == "*" {
		return true
	}

	return matchCommandPattern(pattern, command)
}

func matchCommandPattern(pattern, command string) bool {
	if !hasCommandWildcard(pattern) {
		return unescapeCommandPattern(pattern) == command
	}

	re, err := regexp.Compile("^" + commandPatternRegex(pattern) + "$")
	if err != nil {
		return false
	}

	return re.MatchString(command)
}

func hasCommandWildcard(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '\\' {
			i++

			continue
		}

		if pattern[i] == '*' || pattern[i] == '?' {
			return true
		}
	}

	return false
}

func unescapeCommandPattern(pattern string) string {
	var out strings.Builder
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '\\' && i+1 < len(pattern) {
			out.WriteByte(pattern[i+1])
			i++

			continue
		}

		out.WriteByte(pattern[i])
	}

	return out.String()
}

func commandPatternRegex(pattern string) string {
	var out strings.Builder
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '\\' && i+1 < len(pattern) {
			out.WriteString(regexp.QuoteMeta(pattern[i+1 : i+2]))
			i++

			continue
		}

		switch pattern[i] {
		case '*':
			out.WriteString(".*")
		case '?':
			out.WriteByte('.')
		default:
			out.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
		}
	}

	return out.String()
}

// analyzeBashCommand parses command and returns the simple commands it runs
// (for per-part rule matching) along with whether it contains command or
// process substitution ($(...), `...`, <(...), >(...)) — code whose contents
// should not be auto-allowed by broad per-part rules. Substitution contents are
// still included in parts so deny and ask rules can see them. An unparseable
// command is reported as containing substitution so it can never be auto-allowed
// per part.
func analyzeBashCommand(command string) (parts []string, hasSubstitution bool) {
	file, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, true
	}

	syntax.Walk(file, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.CmdSubst:
			hasSubstitution = true
		case *syntax.ProcSubst:
			hasSubstitution = true
		case *syntax.Stmt:
			if call, ok := n.Cmd.(*syntax.CallExpr); ok && len(call.Args) > 0 {
				parts = appendCommandPart(parts, stmtSource(command, n))
			}
		}

		return true
	})

	return parts, hasSubstitution
}

// stmtSource returns the source text of a statement's command and its
// redirections, excluding the trailing separator (;, &, |&) that Stmt.End
// would include.
func stmtSource(command string, stmt *syntax.Stmt) string {
	start := int(stmt.Pos().Offset())
	end := int(stmt.Cmd.End().Offset())

	for _, redir := range stmt.Redirs {
		if e := int(redir.End().Offset()); e > end {
			end = e
		}
	}

	return command[start:end]
}

func appendCommandPart(parts []string, part string) []string {
	part = strings.TrimSpace(part)
	if part == "" {
		return parts
	}

	return append(parts, part)
}
