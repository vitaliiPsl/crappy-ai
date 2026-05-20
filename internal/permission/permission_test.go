package permission

import (
	"context"
	"errors"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type memoryStore struct {
	permissions Permissions
	saved       []savedRule
}

type savedRule struct {
	value Decision
	rule  Rule
}

func (s *memoryStore) Load(context.Context) (Permissions, error) {
	return s.permissions, nil
}

func (s *memoryStore) Add(_ context.Context, value Decision, rule Rule) error {
	s.saved = append(s.saved, savedRule{value: value, rule: rule})
	s.permissions.Add(value, rule)

	return nil
}

type handler struct {
	calls int
	resp  Response
	err   error
}

func (h *handler) Ask(context.Context, string, kit.ToolCall) (Response, error) {
	h.calls++

	return h.resp, h.err
}

func readCall(path string) kit.ToolCall {
	return kit.NewToolCall("call_1", "read_file", map[string]any{"path": path})
}

func bashCall(command string) kit.ToolCall {
	return kit.NewToolCall("call_1", "bash", map[string]any{"command": command})
}

func TestService_AllowsConfiguredRule(t *testing.T) {
	store := &memoryStore{
		permissions: Permissions{
			Allow: []Rule{{Tool: "read_file", Pattern: "//tmp/project/**"}},
		},
	}
	h := &handler{err: errors.New("should not ask")}

	err := NewService(store, h).Authorize(context.Background(), "session-1", readCall("/tmp/project/main.go"))
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if h.calls != 0 {
		t.Fatalf("handler calls = %d, want 0", h.calls)
	}
}

func TestService_DeniesConfiguredRule(t *testing.T) {
	store := &memoryStore{
		permissions: Permissions{
			Deny: []Rule{{Tool: "read_file", Pattern: "//etc/**"}},
		},
	}
	h := &handler{resp: Response{Decision: Allow, Scope: ScopeOnce}}

	err := NewService(store, h).Authorize(context.Background(), "session-1", readCall("/etc/passwd"))
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Authorize error = %v, want ErrDenied", err)
	}

	if h.calls != 0 {
		t.Fatalf("handler calls = %d, want 0", h.calls)
	}
}

func TestService_AsksAndSavesGlobalRule(t *testing.T) {
	store := &memoryStore{}
	h := &handler{
		resp: Response{
			Decision: Allow,
			Scope:    ScopeGlobal,
			Pattern:  "//tmp/project/**",
		},
	}
	service := NewService(store, h)

	if err := service.Authorize(context.Background(), "session-1", readCall("/tmp/project/main.go")); err != nil {
		t.Fatalf("first Authorize: %v", err)
	}

	if err := service.Authorize(context.Background(), "session-1", readCall("/tmp/project/main.go")); err != nil {
		t.Fatalf("second Authorize: %v", err)
	}

	if h.calls != 1 {
		t.Fatalf("handler calls = %d, want 1", h.calls)
	}

	if len(store.saved) != 1 {
		t.Fatalf("saved rules = %d, want 1", len(store.saved))
	}

	if got := store.saved[0]; got.value != Allow || got.rule.Tool != "read_file" {
		t.Fatalf("saved rule = %+v, want allow read_file", got)
	}
}

func TestService_OnceIsNotSaved(t *testing.T) {
	store := &memoryStore{}
	h := &handler{resp: Response{Decision: Allow, Scope: ScopeOnce}}

	if err := NewService(store, h).Authorize(context.Background(), "session-1", readCall("/tmp/project/main.go")); err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if len(store.saved) != 0 {
		t.Fatalf("saved rules = %d, want 0", len(store.saved))
	}
}

func TestService_AskDenyReturnsDenied(t *testing.T) {
	store := &memoryStore{}
	h := &handler{
		resp: Response{
			Decision: Deny,
			Scope:    ScopeOnce,
		},
	}

	err := NewService(store, h).Authorize(context.Background(), "session-1", readCall("/tmp/project/main.go"))
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Authorize error = %v, want ErrDenied", err)
	}
}

func TestResolveBashAllowsExactCommand(t *testing.T) {
	perms := Permissions{
		Default: Ask,
		Allow:   []Rule{{Tool: "bash", Pattern: "go test ./internal/permission"}},
	}

	got := Resolve(perms, bashCall("go test ./internal/permission"))
	if got != Allow {
		t.Fatalf("Resolve = %q, want %q", got, Allow)
	}
}

func TestResolveBashAllowsWildcardCommand(t *testing.T) {
	perms := Permissions{
		Default: Ask,
		Allow:   []Rule{{Tool: "bash", Pattern: "go test *"}},
	}

	got := Resolve(perms, bashCall("go test ./internal/permission"))
	if got != Allow {
		t.Fatalf("Resolve = %q, want %q", got, Allow)
	}
}

func TestResolveBashAllowsCompoundOnlyWhenEveryPartAllowed(t *testing.T) {
	perms := Permissions{
		Default: Ask,
		Allow: []Rule{
			{Tool: "bash", Pattern: "go test *"},
			{Tool: "bash", Pattern: "go vet *"},
		},
	}

	got := Resolve(perms, bashCall("go test ./... && go vet ./..."))
	if got != Allow {
		t.Fatalf("Resolve = %q, want %q", got, Allow)
	}
}

func TestResolveBashDoesNotAllowCompoundWhenOnlyOnePartAllowed(t *testing.T) {
	perms := Permissions{
		Default: Ask,
		Allow:   []Rule{{Tool: "bash", Pattern: "echo *"}},
	}

	got := Resolve(perms, bashCall("echo ok && rm -rf tmp"))
	if got != Ask {
		t.Fatalf("Resolve = %q, want %q", got, Ask)
	}
}

func TestResolveBashDenyMatchesCompoundPart(t *testing.T) {
	perms := Permissions{
		Default: Ask,
		Deny:    []Rule{{Tool: "bash", Pattern: "rm *"}},
		Allow:   []Rule{{Tool: "bash"}},
	}

	got := Resolve(perms, bashCall("echo ok && rm -rf tmp"))
	if got != Deny {
		t.Fatalf("Resolve = %q, want %q", got, Deny)
	}
}

func TestResolveBashSubstitutionSkipsAllow(t *testing.T) {
	perms := Permissions{
		Default: Ask,
		Allow:   []Rule{{Tool: "bash", Pattern: "git *"}},
	}

	got := Resolve(perms, bashCall("git $(rm -rf /)"))
	if got != Ask {
		t.Fatalf("Resolve = %q, want %q", got, Ask)
	}
}

func TestResolveBashSubstitutionStillDenied(t *testing.T) {
	perms := Permissions{
		Default: Ask,
		Deny:    []Rule{{Tool: "bash", Pattern: "git *"}},
	}

	got := Resolve(perms, bashCall("git $(rm -rf /)"))
	if got != Deny {
		t.Fatalf("Resolve = %q, want %q", got, Deny)
	}
}

func TestResolveBashDenyMatchesCommandInsideSubstitution(t *testing.T) {
	perms := Permissions{
		Default: Ask,
		Deny:    []Rule{{Tool: "bash", Pattern: "rm *"}},
		Allow:   []Rule{{Tool: "bash"}},
	}

	got := Resolve(perms, bashCall("echo $(rm -rf /)"))
	if got != Deny {
		t.Fatalf("Resolve = %q, want %q", got, Deny)
	}
}

func TestResolveBashAskMatchesCommandInsideProcessSubstitution(t *testing.T) {
	perms := Permissions{
		Default: Allow,
		Ask:     []Rule{{Tool: "bash", Pattern: "ls *"}},
	}

	got := Resolve(perms, bashCall("diff <(ls a) <(ls b)"))
	if got != Ask {
		t.Fatalf("Resolve = %q, want %q", got, Ask)
	}
}

func TestResolveBashAllowsExactCompoundCommand(t *testing.T) {
	command := "echo ok && rm -rf tmp"
	perms := Permissions{
		Default: Ask,
		Allow:   []Rule{{Tool: "bash", Pattern: command}},
	}

	got := Resolve(perms, bashCall(command))
	if got != Allow {
		t.Fatalf("Resolve = %q, want %q", got, Allow)
	}
}
