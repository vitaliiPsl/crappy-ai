package planning

import (
	"github.com/vitaliiPsl/crappy-adk/agent"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
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

func New(store session.ArtifactStore) extension.Extension {
	return &ext{store: store}
}

func (e *ext) Name() string {
	return "planning"
}

func (e *ext) Options(ctx extension.Context) (agent.Option, error) {
	return agent.WithExtensions(
		agent.WithInstructions(Instructions),
		agent.WithTools(newTool(ctx.SessionID, e.store)),
		agent.WithOnModelRequest(injectPlan(ctx.SessionID, e.store)),
	), nil
}
