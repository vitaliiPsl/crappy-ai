package factory

import (
	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/guard"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/instructions"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/tools"
)

const (
	coreSource         = "core"
	toolLoopMaxRepeats = 3
	toolLoopWindow     = 5
)

type coreContributor struct {
	permissions *permission.Service
	background  *background.Manager
}

func (e coreContributor) Name() string {
	return coreSource
}

func (e coreContributor) Spec(ctx Context) (spec.AgentSpec, error) {
	return spec.AgentSpec{
		Context: e.context(ctx),
		Tools:   e.tools(ctx),
		Hooks:   e.hooks(ctx),
	}, nil
}

func (e coreContributor) context(ctx Context) []spec.ContextPiece {
	return []spec.ContextPiece{
		{
			Name:    "System prompt",
			Source:  coreSource,
			Kind:    spec.ContextSystemPrompt,
			Content: ctx.Config.Prompt,
		},
		{
			Name:    "Environment",
			Source:  coreSource,
			Kind:    spec.ContextEnvironment,
			Content: instructions.Env(ctx.Config.Cwd),
		},
		{
			Name:    "Instruction files",
			Source:  coreSource,
			Kind:    spec.ContextInstructions,
			Content: instructions.Files(ctx.Config.Cwd),
		},
	}
}

func (e coreContributor) tools(ctx Context) []spec.ToolSpec {
	coreTools := tools.Core(e.background.ForSession(ctx.SessionID))

	out := make([]spec.ToolSpec, 0, len(coreTools))
	for _, t := range coreTools {
		out = append(out, spec.ToolSpec{
			Source: coreSource,
			Tool:   t,
		})
	}

	return out
}

func (e coreContributor) hooks(ctx Context) []spec.HookSpec {
	return []spec.HookSpec{
		e.permissionHook(ctx.SessionID),
		e.repeatedToolCallHook(toolLoopMaxRepeats, toolLoopWindow),
	}
}

func (e coreContributor) permissionHook(sessionID string) spec.HookSpec {
	return spec.HookSpec{
		Name:   "Permission enforcement",
		Source: coreSource,
		Kind:   spec.HookToolCall,
		Option: agent.WithOnToolCall(func(rc *kit.RunContext, call kit.ToolCall) (kit.ToolCall, error) {
			if err := e.permissions.Authorize(rc.Context, sessionID, call); err != nil {
				return call, err
			}

			return call, nil
		}),
	}
}

func (e coreContributor) repeatedToolCallHook(maxRepeats, window int) spec.HookSpec {
	return spec.HookSpec{
		Name:   "Repeated tool call limit",
		Source: coreSource,
		Kind:   spec.HookModelResponse,
		Option: guard.WithRepeatedToolCallLimit(maxRepeats, window),
	}
}
