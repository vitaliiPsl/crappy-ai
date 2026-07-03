package permission

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/strategy"
)

type asker struct {
	calls    int
	requests []ask.Request
	resp     ask.Response
	err      error
}

func (a *asker) Ask(_ context.Context, request ask.Request) (ask.Response, error) {
	a.calls++
	a.requests = append(a.requests, request)

	return a.resp, a.err
}

func readCall(path string) kit.ToolCall {
	return kit.NewToolCall("call_1", strategy.ToolReadFile, map[string]any{"path": path})
}

func testConfigStore(t *testing.T, permissions model.Permissions) *config.Store {
	t.Helper()

	permissions.Default = model.Ask

	return config.NewStore(
		config.Config{Mode: config.ModeDefault, Agent: config.Agent{Permissions: permissions}},
		filepath.Join(t.TempDir(), "config.yaml"),
	)
}

func testConfigStoreWithConfig(t *testing.T, cfg config.Config) *config.Store {
	t.Helper()

	return config.NewStore(cfg, filepath.Join(t.TempDir(), "config.yaml"))
}

func testConfig(store *config.Store) config.Config {
	cfg := store.Get()

	return cfg
}

func testContext(store *config.Store, a ask.Asker) Context {
	cfg := testConfig(store)

	return Context{
		Mode:        cfg.Mode,
		Permissions: cfg.Permissions,
		Asker:       a,
	}
}

func TestService_AllowsConfiguredRule(t *testing.T) {
	configStore := testConfigStore(t, model.Permissions{
		Allow: []model.Rule{{Tool: strategy.ToolReadFile, Pattern: "//tmp/project/**"}},
	})
	a := &asker{err: errors.New("should not ask")}

	err := NewService(configStore).Authorize(context.Background(), testContext(configStore, a), readCall("/tmp/project/main.go"))
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if a.calls != 0 {
		t.Fatalf("asker calls = %d, want 0", a.calls)
	}
}

func TestService_DeniesConfiguredRule(t *testing.T) {
	configStore := testConfigStore(t, model.Permissions{
		Deny: []model.Rule{{Tool: strategy.ToolReadFile, Pattern: "//etc/**"}},
	})
	a := &asker{resp: ask.Response{OptionID: model.OptionAllowOnce}}

	err := NewService(configStore).Authorize(context.Background(), testContext(configStore, a), readCall("/etc/passwd"))
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Authorize error = %v, want ErrDenied", err)
	}

	if a.calls != 0 {
		t.Fatalf("asker calls = %d, want 0", a.calls)
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
	a := &asker{err: errors.New("should not ask")}

	err := NewService(configStore).Authorize(context.Background(), testContext(configStore, a), readCall("/etc/passwd"))
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if a.calls != 0 {
		t.Fatalf("asker calls = %d, want 0", a.calls)
	}
}

func TestService_InvalidModeReturnsError(t *testing.T) {
	configStore := testConfigStoreWithConfig(t, config.Config{
		Mode: config.Mode("warp"),
	})
	a := &asker{resp: ask.Response{OptionID: model.OptionAllowOnce}}

	err := NewService(configStore).Authorize(context.Background(), testContext(configStore, a), readCall("/tmp/project/main.go"))
	if err == nil {
		t.Fatal("Authorize error = nil, want invalid mode error")
	}

	if a.calls != 0 {
		t.Fatalf("asker calls = %d, want 0", a.calls)
	}
}

func TestService_AsksAndSavesSelectedGlobalRule(t *testing.T) {
	configStore := testConfigStore(t, model.Permissions{})
	a := &asker{
		resp: ask.Response{OptionID: model.OptionAllowPattern},
	}
	service := NewService(configStore)

	if err := service.Authorize(context.Background(), testContext(configStore, a), readCall("/tmp/project/main.go")); err != nil {
		t.Fatalf("first Authorize: %v", err)
	}

	if err := service.Authorize(context.Background(), testContext(configStore, a), readCall("/tmp/project/main.go")); err != nil {
		t.Fatalf("second Authorize: %v", err)
	}

	if a.calls != 1 {
		t.Fatalf("asker calls = %d, want 1", a.calls)
	}

	if len(a.requests) != 1 {
		t.Fatalf("ask requests = %d, want 1", len(a.requests))
	}

	req := a.requests[0]
	if req.ID != "call_1" || req.Title != "Allow read_file: /tmp/project/main.go?" || req.Detail != "/tmp/project/main.go" {
		t.Fatalf("ask request = %+v, want read_file prompt for path", req)
	}

	if _, ok := askOption(req, model.OptionAllowPattern); !ok {
		t.Fatalf("ask options = %+v, want allow pattern option", req.Options)
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
	a := &asker{resp: ask.Response{OptionID: model.OptionAllowOnce}}

	if err := NewService(configStore).Authorize(context.Background(), testContext(configStore, a), readCall("/tmp/project/main.go")); err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if got := configStore.Get().Permissions; len(got.Allow) != 0 || len(got.Ask) != 0 || len(got.Deny) != 0 {
		t.Fatalf("saved permissions = %+v, want none", got)
	}
}

func TestService_AskDenyReturnsDenied(t *testing.T) {
	configStore := testConfigStore(t, model.Permissions{})
	a := &asker{
		resp: ask.Response{OptionID: model.OptionDenyOnce},
	}

	err := NewService(configStore).Authorize(context.Background(), testContext(configStore, a), readCall("/tmp/project/main.go"))
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Authorize error = %v, want ErrDenied", err)
	}
}

func askOption(req ask.Request, id string) (ask.Option, bool) {
	for _, option := range req.Options {
		if option.ID == id {
			return option, true
		}
	}

	return ask.Option{}, false
}

func TestService_AskWithoutAskerReturnsDenied(t *testing.T) {
	configStore := testConfigStore(t, model.Permissions{})

	err := NewService(configStore).Authorize(context.Background(), testContext(configStore, nil), readCall("/tmp/project/main.go"))
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Authorize error = %v, want ErrDenied", err)
	}
}
