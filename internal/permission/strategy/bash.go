package strategy

import (
	"regexp"
	"strings"

	"mvdan.cc/sh/v3/syntax"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
)

type bashStrategy struct{}

var (
	commandNamePattern       = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)
	commandSubcommandPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
)

var unsafePatternCommands = map[string]bool{
	"sh":         true,
	"bash":       true,
	"zsh":        true,
	"fish":       true,
	"csh":        true,
	"tcsh":       true,
	"ksh":        true,
	"dash":       true,
	"cmd":        true,
	"powershell": true,
	"pwsh":       true,
	"env":        true,
	"xargs":      true,
	"nice":       true,
	"stdbuf":     true,
	"nohup":      true,
	"timeout":    true,
	"time":       true,
	"sudo":       true,
	"doas":       true,
	"pkexec":     true,
}

func (bashStrategy) Resolve(permissions model.Permissions, call kit.ToolCall) model.ResolveResult {
	input := extractInput(call)
	command := strings.TrimSpace(input)
	parts, hasSubstitution := parseBashCommand(command)

	for _, rule := range permissions.Deny {
		if bashRuleMatchesAny(rule, command, parts) {
			return resolveResult(model.Deny, call, input, nil)
		}
	}

	options := bashOptions(call.Name, command, parts, hasSubstitution)
	for _, rule := range permissions.Ask {
		if bashRuleMatchesAny(rule, command, parts) {
			return resolveResult(model.Ask, call, input, options)
		}
	}

	if bashAllowed(permissions.Allow, command, parts, hasSubstitution) {
		return resolveResult(model.Allow, call, input, nil)
	}

	return resolveResult(permissions.Default, call, input, options)
}

func bashOptions(tool, command string, parts []string, hasSubstitution bool) []model.Option {
	if command == "" {
		return nil
	}

	options := []model.Option{
		{
			ID:       model.OptionAllowExact,
			Label:    "Allow exact command",
			Decision: model.Allow,
			Scope:    model.ScopeGlobal,
			Rule:     &model.Rule{Tool: tool, Pattern: command},
		},
	}

	if !hasSubstitution {
		if pattern, ok := bashCommandPattern(command, parts); ok {
			options = append(options, model.Option{
				ID:       model.OptionAllowPattern,
				Label:    "Allow command pattern",
				Decision: model.Allow,
				Scope:    model.ScopeGlobal,
				Rule:     &model.Rule{Tool: tool, Pattern: pattern},
			})
		}
	}

	return options
}

func bashCommandPattern(command string, parts []string) (string, bool) {
	if len(parts) != 1 || parts[0] != command {
		return "", false
	}

	fields := strings.Fields(command)
	if len(fields) < 2 {
		return "", false
	}

	if !commandNamePattern.MatchString(fields[0]) {
		return "", false
	}

	if unsafePatternCommands[fields[0]] {
		return "", false
	}

	if !commandSubcommandPattern.MatchString(fields[1]) {
		return "", false
	}

	return fields[0] + " " + fields[1] + " *", true
}

func bashAllowed(rules []model.Rule, command string, parts []string, hasSubstitution bool) bool {
	for _, rule := range rules {
		if rule.Tool != ToolBash {
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

func bashPartAllowed(rules []model.Rule, command string) bool {
	for _, rule := range rules {
		if rule.Tool == ToolBash && matchBash(rule.Pattern, command) {
			return true
		}
	}

	return false
}

func bashRuleMatchesAny(rule model.Rule, command string, parts []string) bool {
	if rule.Tool != ToolBash {
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
		return pattern == command
	}

	re, err := regexp.Compile("^" + commandPatternRegex(pattern) + "$")
	if err != nil {
		return false
	}

	return re.MatchString(command)
}

func hasCommandWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?")
}

func commandPatternRegex(pattern string) string {
	var out strings.Builder
	for i := 0; i < len(pattern); i++ {
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

// parseBashCommand parses command and returns the simple commands it runs
// (for per-part rule matching) along with whether it contains command or
// process substitution ($(...), `...`, <(...), >(...)) — code whose contents
// should not be auto-allowed by broad per-part rules. Substitution contents are
// still included in parts so deny and ask rules can see them. An unparseable
// command is reported as containing substitution so it can never be auto-allowed
// per part.
func parseBashCommand(command string) (parts []string, hasSubstitution bool) {
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
