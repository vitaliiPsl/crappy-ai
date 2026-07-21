package memory

import (
	"context"

	adk "github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
	corememory "github.com/vitaliiPsl/crappy-ai/internal/memory"
)

const instructions = `# Persistent memory

You can manage persistent memories about the user with memory tools.

- Only remember information when the user explicitly asks you to remember it.
- Store one concise, durable memory at a time.
- Use profile for stable facts, preference for user preferences, and instruction for explicitly requested persistent behavior.
- Never store passwords, API keys, tokens, or other secrets.
- Use memory_list before updating or forgetting so you have the exact memory ID.
- Persistent memories may be outdated. Current user instructions and directly observed evidence take precedence.
- AGENTS.md and CLAUDE.md are user-owned instruction files. Never edit them to record memories.`

type ext struct {
	store corememory.Store
}

var _ appagent.Contributor = (*ext)(nil)

func New(store corememory.Store) appagent.Contributor {
	return &ext{store: store}
}

func (e *ext) Contribute(_ context.Context, _ appagent.Request) (appagent.Contribution, error) {
	if e.store == nil {
		return appagent.Contribution{}, nil
	}

	return appagent.Contribution{
		Tools: newTools(e.store),
		Options: []adk.Option{
			adk.WithInstructions(instructions),
			adk.WithDynamicInstructions(func(rc *kit.RunContext) (string, error) {
				memories, err := e.store.List(rc.Context)
				if err != nil {
					return "", err
				}

				return formatContext(memories), nil
			}),
		},
	}, nil
}
