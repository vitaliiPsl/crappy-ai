package strategy

import (
	"slices"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
)

func bashCall(command string) kit.ToolCall {
	return kit.NewToolCall("call_1", ToolBash, map[string]any{inputCommand: command})
}

func TestParseBashCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		wantParts []string
		wantSubst bool
	}{
		{
			name:      "single command",
			command:   "go test ./...",
			wantParts: []string{"go test ./..."},
		},
		{
			name:      "and splits into commands",
			command:   "go test ./... && go vet ./...",
			wantParts: []string{"go test ./...", "go vet ./..."},
		},
		{
			name:      "semicolon splits",
			command:   "pwd; ls -la",
			wantParts: []string{"pwd", "ls -la"},
		},
		{
			name:      "pipe splits",
			command:   "cat file | grep needle",
			wantParts: []string{"cat file", "grep needle"},
		},
		{
			name:      "background splits without trailing amp",
			command:   "echo ok & rm -rf tmp",
			wantParts: []string{"echo ok", "rm -rf tmp"},
		},
		{
			name:      "redirection stays attached to command",
			command:   "make 2>&1",
			wantParts: []string{"make 2>&1"},
		},
		{
			name:      "operator inside single quotes is literal",
			command:   "echo 'a && b' && pwd",
			wantParts: []string{"echo 'a && b'", "pwd"},
		},
		{
			name:      "subshell group exposes inner command",
			command:   "(rm -rf /)",
			wantParts: []string{"rm -rf /"},
		},
		{
			name:      "command substitution is flagged",
			command:   "git $(rm -rf /)",
			wantParts: []string{"git $(rm -rf /)", "rm -rf /"},
			wantSubst: true,
		},
		{
			name:      "backtick substitution is flagged",
			command:   "echo `date`",
			wantParts: []string{"echo `date`", "date"},
			wantSubst: true,
		},
		{
			name:      "process substitution is flagged",
			command:   "diff <(ls a) <(ls b)",
			wantParts: []string{"diff <(ls a) <(ls b)", "ls a", "ls b"},
			wantSubst: true,
		},
		{
			name:      "parameter expansion is not substitution",
			command:   "echo ${HOME}",
			wantParts: []string{"echo ${HOME}"},
		},
		{
			name:      "unparseable command reports substitution",
			command:   "echo 'unterminated",
			wantParts: nil,
			wantSubst: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, subst := parseBashCommand(tt.command)

			if !slices.Equal(parts, tt.wantParts) {
				t.Errorf("parts = %#v, want %#v", parts, tt.wantParts)
			}

			if subst != tt.wantSubst {
				t.Errorf("hasSubstitution = %v, want %v", subst, tt.wantSubst)
			}
		})
	}
}

func TestMatchBash(t *testing.T) {
	tests := []struct {
		pattern string
		command string
		want    bool
	}{
		{pattern: "git status", command: "git status", want: true},
		{pattern: "git status", command: "git status --short", want: false},
		{pattern: "go test *", command: "go test ./internal/permission", want: true},
		{pattern: `echo \*`, command: "echo *", want: false},
		{pattern: "echo ?", command: "echo x", want: true},
	}

	for _, tt := range tests {
		got := matchBash(tt.pattern, tt.command)
		if got != tt.want {
			t.Fatalf("matchBash(%q, %q) = %v, want %v", tt.pattern, tt.command, got, tt.want)
		}
	}
}

func TestBashCommandPattern(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
		wantOK  bool
	}{
		{
			name:    "command subcommand",
			command: "go test ./...",
			want:    "go test *",
			wantOK:  true,
		},
		{
			name:    "hyphenated subcommand",
			command: "docker compose-up service",
			want:    "docker compose-up *",
			wantOK:  true,
		},
		{
			name:    "flag is not subcommand",
			command: "ls -la",
		},
		{
			name:    "file is not subcommand",
			command: "cat doc.md",
		},
		{
			name:    "number is not subcommand",
			command: "chmod 755 file",
		},
		{
			name:    "path command is not suggested",
			command: "./script run thing",
		},
		{
			name:    "env assignment is not suggested",
			command: "NODE_ENV=test npm run build",
		},
		{
			name:    "shell wrapper is not suggested",
			command: "bash run thing",
		},
		{
			name:    "privilege wrapper is not suggested",
			command: "sudo npm run build",
		},
		{
			name:    "compound command is not suggested",
			command: "go test ./... && go vet ./...",
		},
		{
			name:    "empty command is not suggested",
			command: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := strings.TrimSpace(tt.command)
			parts, _ := parseBashCommand(command)

			got, ok := bashCommandPattern(command, parts)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}

			if got != tt.want {
				t.Fatalf("pattern = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAskRequestBashRules(t *testing.T) {
	request := askRequest(t, bashCall("go test ./..."))

	exact := optionRule(t, request, model.OptionAllowExact)
	if exact != (model.Rule{Tool: ToolBash, Pattern: "go test ./..."}) {
		t.Fatalf("exact rule = %+v, want exact bash command", exact)
	}

	pattern := optionRule(t, request, model.OptionAllowPattern)
	if pattern != (model.Rule{Tool: ToolBash, Pattern: "go test *"}) {
		t.Fatalf("pattern rule = %+v, want bash command pattern", pattern)
	}
}

func TestAskRequestBashCompoundDoesNotSuggestPattern(t *testing.T) {
	request := askRequest(t, bashCall("go test ./... && go vet ./..."))

	if _, ok := request.Option(model.OptionAllowPattern); ok {
		t.Fatalf("compound bash request unexpectedly has pattern option: %#v", request.Options)
	}
}

func TestAskRequestBashSubstitutionDoesNotSuggestPattern(t *testing.T) {
	request := askRequest(t, bashCall("go test $(pwd)"))

	if _, ok := request.Option(model.OptionAllowPattern); ok {
		t.Fatalf("bash request with substitution unexpectedly has pattern option: %#v", request.Options)
	}
}

func TestResolveBashAllowsExactCommand(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Allow:   []model.Rule{{Tool: ToolBash, Pattern: "go test ./internal/permission"}},
	}

	got := Resolve(perms, bashCall("go test ./internal/permission")).Decision
	if got != model.Allow {
		t.Fatalf("Resolve = %q, want %q", got, model.Allow)
	}
}

func TestResolveBashAllowsWildcardCommand(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Allow:   []model.Rule{{Tool: ToolBash, Pattern: "go test *"}},
	}

	got := Resolve(perms, bashCall("go test ./internal/permission")).Decision
	if got != model.Allow {
		t.Fatalf("Resolve = %q, want %q", got, model.Allow)
	}
}

func TestResolveBashAllowsCompoundOnlyWhenEveryPartAllowed(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Allow: []model.Rule{
			{Tool: ToolBash, Pattern: "go test *"},
			{Tool: ToolBash, Pattern: "go vet *"},
		},
	}

	got := Resolve(perms, bashCall("go test ./... && go vet ./...")).Decision
	if got != model.Allow {
		t.Fatalf("Resolve = %q, want %q", got, model.Allow)
	}
}

func TestResolveBashDoesNotAllowCompoundWhenOnlyOnePartAllowed(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Allow:   []model.Rule{{Tool: ToolBash, Pattern: "echo *"}},
	}

	got := Resolve(perms, bashCall("echo ok && rm -rf tmp")).Decision
	if got != model.Ask {
		t.Fatalf("Resolve = %q, want %q", got, model.Ask)
	}
}

func TestResolveBashDenyMatchesCompoundPart(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Deny:    []model.Rule{{Tool: ToolBash, Pattern: "rm *"}},
		Allow:   []model.Rule{{Tool: ToolBash}},
	}

	got := Resolve(perms, bashCall("echo ok && rm -rf tmp")).Decision
	if got != model.Deny {
		t.Fatalf("Resolve = %q, want %q", got, model.Deny)
	}
}

func TestResolveBashSubstitutionSkipsAllow(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Allow:   []model.Rule{{Tool: ToolBash, Pattern: "git *"}},
	}

	got := Resolve(perms, bashCall("git $(rm -rf /)")).Decision
	if got != model.Ask {
		t.Fatalf("Resolve = %q, want %q", got, model.Ask)
	}
}

func TestResolveBashSubstitutionStillDenied(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Deny:    []model.Rule{{Tool: ToolBash, Pattern: "git *"}},
	}

	got := Resolve(perms, bashCall("git $(rm -rf /)")).Decision
	if got != model.Deny {
		t.Fatalf("Resolve = %q, want %q", got, model.Deny)
	}
}

func TestResolveBashDenyMatchesCommandInsideSubstitution(t *testing.T) {
	perms := model.Permissions{
		Default: model.Ask,
		Deny:    []model.Rule{{Tool: ToolBash, Pattern: "rm *"}},
		Allow:   []model.Rule{{Tool: ToolBash}},
	}

	got := Resolve(perms, bashCall("echo $(rm -rf /)")).Decision
	if got != model.Deny {
		t.Fatalf("Resolve = %q, want %q", got, model.Deny)
	}
}

func TestResolveBashAskMatchesCommandInsideProcessSubstitution(t *testing.T) {
	perms := model.Permissions{
		Default: model.Allow,
		Ask:     []model.Rule{{Tool: ToolBash, Pattern: "ls *"}},
	}

	got := Resolve(perms, bashCall("diff <(ls a) <(ls b)")).Decision
	if got != model.Ask {
		t.Fatalf("Resolve = %q, want %q", got, model.Ask)
	}
}

func TestResolveBashAllowsExactCompoundCommand(t *testing.T) {
	command := "echo ok && rm -rf tmp"
	perms := model.Permissions{
		Default: model.Ask,
		Allow:   []model.Rule{{Tool: ToolBash, Pattern: command}},
	}

	got := Resolve(perms, bashCall(command)).Decision
	if got != model.Allow {
		t.Fatalf("Resolve = %q, want %q", got, model.Allow)
	}
}
