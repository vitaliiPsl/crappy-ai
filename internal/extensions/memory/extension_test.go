package memory

import (
	"context"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/kittest"
	xmemory "github.com/vitaliiPsl/crappy-adk/x/memory"
	xtool "github.com/vitaliiPsl/crappy-adk/x/tool"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
	corememory "github.com/vitaliiPsl/crappy-ai/internal/memory"
)

func TestExtensionAddsMemoryContextAndTools(t *testing.T) {
	store := &fakeStore{memories: []corememory.Memory{
		{ID: "memory-1", Kind: corememory.KindPreference, Content: "Prefers concise answers."},
	}}
	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{
			Message:      kit.NewModelMessage(kit.NewTextContent("done")),
			FinishReason: kit.FinishReasonStop,
		},
	})

	contribution, err := New(store).Contribute(context.Background(), appagent.Request{})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	ag, err := agent.New(model, xmemory.NewHistory(), xtool.NewSet(contribution.Tools...), contribution.Options...)
	if err != nil {
		t.Fatalf("New agent: %v", err)
	}

	_, err = ag.Run(context.Background(), kit.NewUserMessage(kit.NewTextContent("hello")))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	req := model.CallAt(0)
	for _, want := range []string{
		"# Memory policy",
		"# Saved memories",
		"## Preferences",
		"- Prefers concise answers.",
		"proactively when the user directly reveals something durable",
		"Save useful memories directly",
		"Never infer an instruction.",
		"Never edit them to record memories.",
	} {
		if !strings.Contains(req.Instructions, want) {
			t.Fatalf("instructions missing %q:\n%s", want, req.Instructions)
		}
	}

	if len(req.Tools) != 4 {
		t.Fatalf("tools = %d, want 4", len(req.Tools))
	}
}

func TestExtensionNoopsWithoutStore(t *testing.T) {
	contribution, err := New(nil).Contribute(context.Background(), appagent.Request{})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	if len(contribution.Tools) != 0 || len(contribution.Options) != 0 {
		t.Fatalf("contribution = %+v, want empty", contribution)
	}
}
