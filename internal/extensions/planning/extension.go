package planning

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

const Instructions = `# Planning

You have a write_plan tool to create and track task lists for multi-step work.

## When to use it
- The task requires 3 or more distinct steps.
- The user provides multiple things to do at once.
- You need to coordinate changes across several files.

## When NOT to use it
- Simple questions, single-file edits, or quick lookups.
- The task can be completed in one or two straightforward steps.

## How to use it
- Keep tasks concrete and actionable.
- Keep exactly one item "in_progress" at a time.
- Every update replaces the full plan, so send the complete list each time.
- Mark items "completed" only when fully done, and revise the plan as you learn more.`

type ext struct {
	store session.ArtifactStore
}

func New(store session.ArtifactStore) factory.Extension {
	return &ext{store: store}
}

func (e *ext) Name() string {
	return "planning"
}

func (e *ext) Options(_ context.Context, req factory.BuildRequest) ([]kit.Tool, []agent.Option, error) {
	t := newTool(req.SessionID, e.store)

	options := []agent.Option{
		agent.WithInstructions(Instructions),
		agent.WithDynamicInstructions(func(rc *kit.RunContext) (string, error) {
			return currentPlanText(rc.Context, req.SessionID, e.store)
		}),
	}

	return []kit.Tool{t}, options, nil
}
