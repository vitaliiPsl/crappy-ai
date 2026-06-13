package assistant

import (
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/guard"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/instructions"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
	"github.com/vitaliiPsl/crappy-ai/internal/tools"
)

const (
	toolLoopMaxRepeats = 3
	toolLoopWindow     = 5
)

func (a *Assistant) buildAgentSpec(ctx extension.Context) (spec.AgentSpec, error) {
	cfg := ctx.Config

	runSpec := spec.AgentSpec{
		Context: []spec.ContextPiece{
			{
				Name:    "System prompt",
				Source:  "core",
				Kind:    spec.ContextSystemPrompt,
				Content: cfg.SystemPrompt,
			},
			{
				Name:    "Environment",
				Source:  "core",
				Kind:    spec.ContextEnvironment,
				Content: instructions.Env(cfg.Cwd),
			},
			{
				Name:    "Instruction files",
				Source:  "core",
				Kind:    spec.ContextInstructions,
				Content: instructions.Files(cfg.Cwd),
			},
		},
		Tools: coreToolSpecs(tools.Core(a.background.ForSession(ctx.SessionID))),
		Hooks: []spec.HookSpec{
			a.permissionHook(ctx.SessionID),
			a.repeatedToolCallHook(toolLoopMaxRepeats, toolLoopWindow),
		},
	}

	for _, ext := range a.extensions {
		extSpec, err := ext.Spec(ctx)
		if err != nil {
			return spec.AgentSpec{}, fmt.Errorf("extension %q spec: %w", ext.Name(), err)
		}

		runSpec.Merge(extSpec)
	}

	return runSpec, nil
}

func coreToolSpecs(tools []kit.Tool) []spec.ToolSpec {
	out := make([]spec.ToolSpec, 0, len(tools))
	for _, t := range tools {
		out = append(out, spec.ToolSpec{
			Source: "core",
			Tool:   t,
		})
	}

	return out
}

func (a *Assistant) permissionHook(sessionID string) spec.HookSpec {
	return spec.HookSpec{
		Name:   "Permission enforcement",
		Source: "core",
		Kind:   spec.HookToolCall,
		Option: agent.WithOnToolCall(func(rc *kit.RunContext, call kit.ToolCall) (kit.ToolCall, error) {
			if err := a.permissions.Authorize(rc.Context, sessionID, call); err != nil {
				return call, err
			}

			return call, nil
		}),
	}
}

func (a *Assistant) repeatedToolCallHook(maxRepeats, window int) spec.HookSpec {
	return spec.HookSpec{
		Name:   "Repeated tool call limit",
		Source: "core",
		Kind:   spec.HookModelResponse,
		Option: guard.WithRepeatedToolCallLimit(maxRepeats, window),
	}
}
