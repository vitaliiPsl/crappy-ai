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

func testConfigStore(t *testing.T, permissions model.Permissions) *config.Store {
	t.Helper()

	return config.NewStore(
		config.Config{Mode: config.ModeDefault, Agent: config.Agent{Permissions: permissions}},
		filepath.Join(t.TempDir(), "config.yaml"),
	)
}

func testConfigStoreWithConfig(t *testing.T, cfg config.Config) *config.Store {
	t.Helper()

	return config.NewStore(cfg, filepath.Join(t.TempDir(), "config.yaml"))
}

func TestService_AllowsConfiguredRule(t *testing.T) {
	configStore := testConfigStore(t, model.Permissions{
		Allow: []model.Rule{{Tool: strategy.ToolReadFile, Pattern: "//tmp/project/**"}},
	})
	h := &handler{err: errors.New("should not ask")}

	err := NewService(configStore, h).Authorize(context.Background(), "session-1", readCall("/tmp/project/main.go"))
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if h.calls != 0 {
		t.Fatalf("handler calls = %d, want 0", h.calls)
	}
}

func TestService_DeniesConfiguredRule(t *testing.T) {
	configStore := testConfigStore(t, model.Permissions{
		Deny: []model.Rule{{Tool: strategy.ToolReadFile, Pattern: "//etc/**"}},
	})
	h := &handler{resp: model.AskResponse{OptionID: model.OptionAllowOnce}}

	err := NewService(configStore, h).Authorize(context.Background(), "session-1", readCall("/etc/passwd"))
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Authorize error = %v, want ErrDenied", err)
	}

	if h.calls != 0 {
		t.Fatalf("handler calls = %d, want 0", h.calls)
	}
}

func TestService_YoloModeAllowsDeniedRuleWithoutPrompt(t *testing.T) {
	configStore := testConfigStoreWithConfig(t, config.Config{
		Mode: config.ModeYolo,
		Agent: config.Agent{
			Permissions: model.Permissions{
				Deny: []model.Rule{{Tool: strategy.ToolReadFile, Pattern: "//etc/**"}},
			},
		},
	})
	h := &handler{err: errors.New("should not ask")}

	err := NewService(configStore, h).Authorize(context.Background(), "session-1", readCall("/etc/passwd"))
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if h.calls != 0 {
		t.Fatalf("handler calls = %d, want 0", h.calls)
	}
}

func TestService_InvalidModeReturnsError(t *testing.T) {
	configStore := testConfigStoreWithConfig(t, config.Config{
		Mode: config.Mode("warp"),
	})
	h := &handler{resp: model.AskResponse{OptionID: model.OptionAllowOnce}}

	err := NewService(configStore, h).Authorize(context.Background(), "session-1", readCall("/tmp/project/main.go"))
	if err == nil {
		t.Fatal("Authorize error = nil, want invalid mode error")
	}

	if h.calls != 0 {
		t.Fatalf("handler calls = %d, want 0", h.calls)
	}
}

func TestService_AsksAndSavesSelectedGlobalRule(t *testing.T) {
	configStore := testConfigStore(t, model.Permissions{})
	h := &handler{
		resp: model.AskResponse{OptionID: model.OptionAllowPattern},
	}
	service := NewService(configStore, h)

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
	configStore := testConfigStore(t, model.Permissions{})
	h := &handler{resp: model.AskResponse{OptionID: model.OptionAllowOnce}}

	if err := NewService(configStore, h).Authorize(context.Background(), "session-1", readCall("/tmp/project/main.go")); err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if got := configStore.Get().Permissions; len(got.Allow) != 0 || len(got.Ask) != 0 || len(got.Deny) != 0 {
		t.Fatalf("saved permissions = %+v, want none", got)
	}
}

func TestService_AskDenyReturnsDenied(t *testing.T) {
	configStore := testConfigStore(t, model.Permissions{})
	h := &handler{
		resp: model.AskResponse{OptionID: model.OptionDenyOnce},
	}

	err := NewService(configStore, h).Authorize(context.Background(), "session-1", readCall("/tmp/project/main.go"))
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Authorize error = %v, want ErrDenied", err)
	}
}
