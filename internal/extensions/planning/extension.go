package planning

import (
	"context"

	adk "github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
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

var _ appagent.Contributor = (*ext)(nil)

func New(store session.ArtifactStore) appagent.Contributor {
	return &ext{
		store: store,
	}
}

func (e *ext) Contribute(_ context.Context, req appagent.Request) (appagent.Contribution, error) {
	return e.contribution(req.Session.ID), nil
}

func (e *ext) contribution(sessionID string) appagent.Contribution {
	t := newTool(sessionID, e.store)

	options := []adk.Option{
		adk.WithInstructions(Instructions),
		adk.WithDynamicInstructions(func(rc *kit.RunContext) (string, error) {
			return currentPlanText(rc.Context, sessionID, e.store)
		}),
	}

	return appagent.Contribution{Tools: []kit.Tool{t}, Options: options}
}
