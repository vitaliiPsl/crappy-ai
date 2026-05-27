package skills

import (
	"github.com/vitaliiPsl/crappy-adk/agent"

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

func New(registry *coreskills.Registry) agent.Option {
	listing := coreskills.FormatListing(registry.GetSkills())
	if listing == "" {
		return agent.WithExtensions()
	}

	return agent.WithExtensions(
		agent.WithInstructions(instructions+"\n"+listing),
		agent.WithTools(newTool(registry)),
	)
}
