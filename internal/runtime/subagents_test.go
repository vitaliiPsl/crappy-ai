package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
)

func TestSubagentsContributorNoAgentsAddsNothing(t *testing.T) {
	c := subagentsContributor{session: bareSession()}

	got, err := c.Contribute(context.Background(), appagent.Request{
		Config: config.Config{},
	})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	if len(got.Tools) != 0 || len(got.Options) != 0 {
		t.Fatalf("contribution = %d tools/%d options, want nothing", len(got.Tools), len(got.Options))
	}
}

func TestSubagentsContributorAddsListingAndTaskTool(t *testing.T) {
	c := subagentsContributor{session: bareSession()}

	got, err := c.Contribute(context.Background(), appagent.Request{
		Config: config.Config{
			Agents: []config.Agent{
				{Name: "explorer", Description: "Read-only explorer."},
			},
		},
	})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	if len(got.Tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(got.Tools))
	}

	if got.Tools[0].Definition().Name != subagentToolName {
		t.Fatalf("tool name = %q, want %q", got.Tools[0].Definition().Name, subagentToolName)
	}

	if len(got.Options) != 1 {
		t.Fatalf("options len = %d, want 1", len(got.Options))
	}

	if listing := subagentListing([]config.Agent{{Name: "explorer", Description: "Read-only explorer."}}); listing != "- explorer: Read-only explorer.\n" {
		t.Fatalf("listing = %q", listing)
	}
}

func TestSubagentTaskToolUnknownSubagent(t *testing.T) {
	s := bareSession()
	s.configStore = config.NewStore(config.Config{
		Agents: []config.Agent{{Name: "explorer"}},
	}, "")

	c := subagentsContributor{session: s}

	got, err := c.Contribute(context.Background(), appagent.Request{
		Config: s.configStore.Get(),
	})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	if len(got.Tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(got.Tools))
	}

	_, err = got.Tools[0].Execute(kit.NewRunContext(context.Background()), kit.NewToolCall("call-1", "task", map[string]any{
		"agent":       "missing",
		"task":        "do something",
		"description": "find a thing",
	}))
	if err == nil {
		t.Fatal("Execute error = nil, want unknown subagent")
	}

	if !strings.Contains(err.Error(), "unknown subagent") || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error = %q, want unknown subagent missing", err)
	}
}

func TestSubagentTitle(t *testing.T) {
	if got := subagentTitle(SubagentRequest{Agent: "explorer"}); got != "explorer" {
		t.Fatalf("title = %q, want explorer", got)
	}

	if got := subagentTitle(SubagentRequest{Agent: "explorer", Description: "inspect logs"}); got != "explorer: inspect logs" {
		t.Fatalf("title = %q, want explorer: inspect logs", got)
	}
}
