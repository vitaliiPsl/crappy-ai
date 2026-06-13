package planning

import (
	"context"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
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

func (e *ext) Spec(ctx extension.Context) (spec.AgentSpec, error) {
	t := newTool(ctx.SessionID, e.store)

	return spec.AgentSpec{
		Context: []spec.ContextPiece{
			{
				Name:    "Planning instructions",
				Source:  e.Name(),
				Kind:    spec.ContextExtension,
				Content: Instructions,
			},
			{
				Name:   "Current plan",
				Source: e.Name(),
				Kind:   spec.ContextArtifact,
				Resolve: func(runCtx context.Context) (string, error) {
					return currentPlanText(runCtx, ctx.SessionID, e.store)
				},
			},
		},
		Tools: []spec.ToolSpec{
			{
				Source: e.Name(),
				Tool:   t,
			},
		},
	}, nil
}
