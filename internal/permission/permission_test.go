package permission

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/strategy"
)

type handler struct {
	calls    int
	requests []model.AskRequest
	resp     model.AskResponse
	err      error
}

func (h *handler) Ask(_ context.Context, _ string, request model.AskRequest) (model.AskResponse, error) {
	h.calls++
	h.requests = append(h.requests, request)

	return h.resp, h.err
}

func readCall(path string) kit.ToolCall {
	return kit.NewToolCall("call_1", strategy.ToolReadFile, map[string]any{"path": path})
}

func testStore(t *testing.T, permissions model.Permissions) (*Store, *config.Store) {
	t.Helper()

	configStore := config.NewStore(
		config.Config{Permissions: permissions},
		filepath.Join(t.TempDir(), "config.yaml"),
	)

	return NewStore(configStore), configStore
}

func TestService_AllowsConfiguredRule(t *testing.T) {
	store, _ := testStore(t, model.Permissions{
		Allow: []model.Rule{{Tool: strategy.ToolReadFile, Pattern: "//tmp/project/**"}},
	})
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
	store, _ := testStore(t, model.Permissions{
		Deny: []model.Rule{{Tool: strategy.ToolReadFile, Pattern: "//etc/**"}},
	})
	h := &handler{resp: model.AskResponse{OptionID: model.OptionAllowOnce}}

	err := NewService(store, h).Authorize(context.Background(), "session-1", readCall("/etc/passwd"))
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Authorize error = %v, want ErrDenied", err)
	}

	if h.calls != 0 {
		t.Fatalf("handler calls = %d, want 0", h.calls)
	}
}

func TestService_AsksAndSavesSelectedGlobalRule(t *testing.T) {
	store, configStore := testStore(t, model.Permissions{})
	h := &handler{
		resp: model.AskResponse{OptionID: model.OptionAllowPattern},
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

	rules := configStore.Get().Permissions.Allow
	if len(rules) != 1 {
		t.Fatalf("saved allow rules = %d, want 1", len(rules))
	}

	if got := rules[0]; got.Tool != strategy.ToolReadFile || got.Pattern != "//tmp/project/**" {
		t.Fatalf("saved rule = %+v, want allow read_file //tmp/project/**", got)
	}
}

func TestService_OnceIsNotSaved(t *testing.T) {
	store, configStore := testStore(t, model.Permissions{})
	h := &handler{resp: model.AskResponse{OptionID: model.OptionAllowOnce}}

	if err := NewService(store, h).Authorize(context.Background(), "session-1", readCall("/tmp/project/main.go")); err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if got := configStore.Get().Permissions; len(got.Allow) != 0 || len(got.Ask) != 0 || len(got.Deny) != 0 {
		t.Fatalf("saved permissions = %+v, want none", got)
	}
}

func TestService_AskDenyReturnsDenied(t *testing.T) {
	store, _ := testStore(t, model.Permissions{})
	h := &handler{
		resp: model.AskResponse{OptionID: model.OptionDenyOnce},
	}

	err := NewService(store, h).Authorize(context.Background(), "session-1", readCall("/tmp/project/main.go"))
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Authorize error = %v, want ErrDenied", err)
	}
}
