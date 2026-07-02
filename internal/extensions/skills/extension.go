package skills

import (
	"context"

	adk "github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
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

type Extension interface {
	appagent.Contributor
}

var _ appagent.Contributor = (*ext)(nil)

func New(registry *coreskills.Registry) Extension {
	return &ext{
		registry: registry,
	}
}

func (e *ext) Name() string {
	return "skills"
}

func (e *ext) Contribute(context.Context, appagent.Request) (appagent.Contribution, error) {
	return e.contribution(), nil
}

func (e *ext) contribution() appagent.Contribution {
	listing := coreskills.FormatListing(e.registry.GetSkills())
	if listing == "" {
		return appagent.Contribution{}
	}

	t := newTool(e.registry)

	return appagent.Contribution{
		Tools: []kit.Tool{t},
		Options: []adk.Option{
			adk.WithInstructions(instructions + "\n" + listing),
		},
	}
}
