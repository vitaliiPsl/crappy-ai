package skills

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	coreskills "github.com/vitaliiPsl/crappy-ai/internal/skills"
)

const instructions = `# Skills

You have a use_skill tool that loads reusable local workflows.

## When to use it
- A user asks for a task that matches an available skill.
- A user references a slash command such as /review or /explain.

## How to use it
- Call use_skill before answering or working on the matching task.
- Pass the skill name without the leading slash.
- Pass the rest of the user's slash-command text as args.
- After use_skill returns, follow the loaded skill instructions.

Available skills:
`

type ext struct {
	registry *coreskills.Registry
}

func New(registry *coreskills.Registry) factory.Extension {
	return &ext{registry: registry}
}

func (e *ext) Name() string {
	return "skills"
}

func (e *ext) Options(context.Context, factory.BuildRequest) ([]kit.Tool, []agent.Option, error) {
	listing := coreskills.FormatListing(e.registry.GetSkills())
	if listing == "" {
		return nil, nil, nil
	}

	t := newTool(e.registry)

	return []kit.Tool{t}, []agent.Option{
		agent.WithInstructions(instructions + "\n" + listing),
	}, nil
}
