package subagents

import (
	"context"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/kittest"
	xmemory "github.com/vitaliiPsl/crappy-adk/x/memory"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
)

func TestExtensionNoAgentsAddsNothing(t *testing.T) {
	bg := background.NewManager(context.Background())
	defer bg.Close()

	extSpec, err := New(nil, nil, nil, bg).Spec(extension.Context{})
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

	ec := extension.Context{
		Config: config.Config{
			Agents: []config.Agent{
				{Name: "explorer", Description: "Read-only explorer."},
			},
		},
	}

	extSpec, err := New(nil, nil, nil, bg).Spec(ec)
	if err != nil {
		t.Fatalf("Spec: %v", err)
	}

	compiled, err := spec.Compile(extSpec)
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
	ec := extension.Context{
		Config: config.Config{
			Agents: []config.Agent{{Name: "explorer"}},
		},
	}

	_, err := newTool(nil, nil, nil, ec).Execute(kit.NewRunContext(context.Background()), map[string]any{
		"agent": "missing",
		"task":  "do something",
	})
	if err == nil {
		t.Fatal("Execute error = nil, want unknown subagent")
	}

	if !strings.Contains(err.Error(), "unknown subagent") || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error = %q, want unknown subagent %q", err, "missing")
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
