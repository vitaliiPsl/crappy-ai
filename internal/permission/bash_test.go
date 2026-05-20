package permission

import (
	"slices"
	"testing"
)

func TestAnalyzeBashCommand(t *testing.T) {
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
			parts, subst := analyzeBashCommand(tt.command)

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
		{pattern: `echo \*`, command: "echo *", want: true},
		{pattern: "echo ?", command: "echo x", want: true},
	}

	for _, tt := range tests {
		got := matchBash(tt.pattern, tt.command)
		if got != tt.want {
			t.Fatalf("matchBash(%q, %q) = %v, want %v", tt.pattern, tt.command, got, tt.want)
		}
	}
}
