package subagents

import (
	"strings"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
)

const instructions = `# Subagents

You have a task tool that delegates work to a specialized subagent.

## When to use it
- A task matches a subagent's stated purpose.
- You want to isolate exploratory or noisy work from the main conversation.

## How to use it
- Call task with the subagent name and a self-contained description of the work.
- By default the subagent runs to completion and returns its output directly.
- For long-running work, set "background": true to get a job_id and collect the result later with job_result.

Available subagents:
`

type ext struct {
	factory           *factory.Factory
	extensions        []extension.Extension
	modelRegistry     *models.Registry
	backgroundManager *background.Manager
}

func New(agentFactory *factory.Factory, extensions []extension.Extension, modelRegistry *models.Registry, backgroundManager *background.Manager) extension.Extension {
	return &ext{
		factory:           agentFactory,
		extensions:        extensions,
		modelRegistry:     modelRegistry,
		backgroundManager: backgroundManager,
	}
}

func (e *ext) Name() string {
	return "subagents"
}

func (e *ext) Spec(ec extension.Context) (spec.AgentSpec, error) {
	if len(ec.Config.Agents) == 0 {
		return spec.AgentSpec{}, nil
	}

	jobs := e.backgroundManager.ForSession(ec.SessionID)

	tool, err := background.Wrap(newTool(e.factory, e.extensions, e.modelRegistry, ec), jobs)
	if err != nil {
		return spec.AgentSpec{}, err
	}

	return spec.AgentSpec{
		Context: []spec.ContextPiece{
			{
				Name:    "Available subagents",
				Source:  e.Name(),
				Kind:    spec.ContextExtension,
				Content: instructions + listing(ec.Config.Agents),
			},
		},
		Tools: []spec.ToolSpec{
			{
				Source: e.Name(),
				Tool:   tool,
			},
		},
	}, nil
}

func listing(agents []config.Agent) string {
	var b strings.Builder
	for _, a := range agents {
		b.WriteString("- ")
		b.WriteString(a.Name)

		if a.Description != "" {
			b.WriteString(": ")
			b.WriteString(a.Description)
		}

		b.WriteString("\n")
	}

	return b.String()
}
