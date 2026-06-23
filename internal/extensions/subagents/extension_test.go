package subagents

import (
	"context"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/kittest"
	xmemory "github.com/vitaliiPsl/crappy-adk/x/memory"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	sessionstore "github.com/vitaliiPsl/crappy-ai/internal/session/store"
)

func TestExtensionNoAgentsAddsNothing(t *testing.T) {
	bg := background.NewManager(context.Background())
	defer bg.Close()

	extSpec, err := New(nil, nil, nil, bg, nil).Spec(factory.Context{})
	if err != nil {
		t.Fatalf("Spec: %v", err)
	}

	if len(extSpec.Tools) != 0 || len(extSpec.Context) != 0 {
		t.Fatalf("empty catalog produced spec = %+v, want nothing", extSpec)
	}
}

func TestExtensionAddsListingAndTaskTool(t *testing.T) {
	bg := background.NewManager(context.Background())
	defer bg.Close()

	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{
			Message:      kit.NewModelMessage(kit.NewTextContent("done")),
			FinishReason: kit.FinishReasonStop,
		},
	})

	ec := factory.Context{
		Config: config.Config{
			Agents: []config.Agent{
				{Name: "explorer", Description: "Read-only explorer."},
			},
		},
	}

	extSpec, err := New(nil, nil, nil, bg, nil).Spec(ec)
	if err != nil {
		t.Fatalf("Spec: %v", err)
	}

	compiled, err := factory.Compile(extSpec)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	ag, err := agent.New(model, xmemory.NewHistory(), compiled.Tools, compiled.Options...)
	if err != nil {
		t.Fatalf("New agent: %v", err)
	}

	if _, err := ag.Run(context.Background(), kit.NewUserMessage(kit.NewTextContent("delegate"))); err != nil {
		t.Fatalf("Run: %v", err)
	}

	req := model.CallAt(0)
	for _, want := range []string{
		"# Subagents",
		"Available subagents:",
		"- explorer: Read-only explorer.",
	} {
		if !strings.Contains(req.Instructions, want) {
			t.Fatalf("instructions missing %q:\n%s", want, req.Instructions)
		}
	}

	if len(req.Tools) != 1 || req.Tools[0].Definition().Name != toolName {
		t.Fatalf("tools = %#v, want %q", req.Tools, toolName)
	}
}

func TestTaskToolUnknownSubagent(t *testing.T) {
	ec := factory.Context{
		Config: config.Config{
			Agents: []config.Agent{{Name: "explorer"}},
		},
	}

	_, err := (&ext{}).newTool(ec).Execute(kit.NewRunContext(context.Background()), map[string]any{
		"agent":       "missing",
		"description": "find a thing",
		"task":        "do something",
	})
	if err == nil {
		t.Fatal("Execute error = nil, want unknown subagent")
	}

	if !strings.Contains(err.Error(), "unknown subagent") || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error = %q, want unknown subagent %q", err, "missing")
	}
}

func TestRecordUsageAttributesToChildAndParent(t *testing.T) {
	store, err := sessionstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	ctx := context.Background()
	parent, _ := store.Create(ctx, session.CreateParams{Title: "parent"})
	child, _ := store.Create(ctx, session.CreateParams{Title: "child", ParentID: parent.ID})

	e := &ext{sessionStore: store}
	e.recordUsage(kit.NewRunContext(ctx), child.ID, parent.ID, kit.Usage{InputTokens: 100, OutputTokens: 20})

	for _, id := range []string{child.ID, parent.ID} {
		sess, err := store.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get %s: %v", id, err)
		}

		if sess.Usage.InputTokens != 100 || sess.Usage.OutputTokens != 20 {
			t.Fatalf("session %s usage = %+v, want input 100 output 20", id, sess.Usage)
		}
	}
}

func TestListingFormatsAgents(t *testing.T) {
	got := listing([]config.Agent{
		{Name: "explorer", Description: "Read-only explorer."},
		{Name: "builder"},
	})

	want := "- explorer: Read-only explorer.\n- builder\n"
	if got != want {
		t.Fatalf("listing = %q, want %q", got, want)
	}
}
