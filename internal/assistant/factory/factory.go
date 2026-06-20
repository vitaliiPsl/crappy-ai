package factory

import (
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/guard"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/instructions"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/tools"
)

const (
	toolLoopMaxRepeats = 3
	toolLoopWindow     = 5
)

type Factory struct {
	permissions *permission.Service
	background  *background.Manager
}

func New(permissions *permission.Service, bg *background.Manager) *Factory {
	return &Factory{
		permissions: permissions,
		background:  bg,
	}
}

func (f *Factory) Build(ec extension.Context, extensions []extension.Extension, mem kit.Memory) (*agent.Agent, error) {
	runSpec, err := f.spec(ec, extensions)
	if err != nil {
		return nil, fmt.Errorf("build agent spec: %w", err)
	}

	compiled, err := spec.Compile(runSpec)
	if err != nil {
		return nil, fmt.Errorf("compile agent spec: %w", err)
	}

	if ec.Config.Thinking != "" {
		compiled.Options = append(compiled.Options, agent.WithThinking(kit.ThinkingLevel(ec.Config.Thinking)))
	}

	ag, err := agent.New(ec.Model, mem, compiled.Tools, compiled.Options...)
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	return ag, nil
}

func (f *Factory) spec(ec extension.Context, extensions []extension.Extension) (spec.AgentSpec, error) {
	cfg := ec.Config

	runSpec := spec.AgentSpec{
		Context: []spec.ContextPiece{
			{
				Name:    "System prompt",
				Source:  "core",
				Kind:    spec.ContextSystemPrompt,
				Content: cfg.Prompt,
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
		Tools: f.coreToolSpecs(tools.Core(f.background.ForSession(ec.SessionID))),
		Hooks: []spec.HookSpec{
			f.permissionHook(ec.SessionID),
			f.repeatedToolCallHook(toolLoopMaxRepeats, toolLoopWindow),
		},
	}

	for _, ext := range extensions {
		extSpec, err := ext.Spec(ec)
		if err != nil {
			return spec.AgentSpec{}, fmt.Errorf("extension %q spec: %w", ext.Name(), err)
		}

		runSpec.Merge(extSpec)
	}

	runSpec.Tools = allowedTools(runSpec.Tools, cfg.Tools)

	return runSpec, nil
}

func (f *Factory) coreToolSpecs(tools []kit.Tool) []spec.ToolSpec {
	out := make([]spec.ToolSpec, 0, len(tools))
	for _, t := range tools {
		out = append(out, spec.ToolSpec{
			Source: "core",
			Tool:   t,
		})
	}

	return out
}

func (f *Factory) permissionHook(sessionID string) spec.HookSpec {
	return spec.HookSpec{
		Name:   "Permission enforcement",
		Source: "core",
		Kind:   spec.HookToolCall,
		Option: agent.WithOnToolCall(func(rc *kit.RunContext, call kit.ToolCall) (kit.ToolCall, error) {
			if err := f.permissions.Authorize(rc.Context, sessionID, call); err != nil {
				return call, err
			}

			return call, nil
		}),
	}
}

func (f *Factory) repeatedToolCallHook(maxRepeats, window int) spec.HookSpec {
	return spec.HookSpec{
		Name:   "Repeated tool call limit",
		Source: "core",
		Kind:   spec.HookModelResponse,
		Option: guard.WithRepeatedToolCallLimit(maxRepeats, window),
	}
}

func allowedTools(tools []spec.ToolSpec, allow []string) []spec.ToolSpec {
	if len(allow) == 0 {
		return tools
	}

	allowed := make(map[string]struct{}, len(allow))
	for _, name := range allow {
		allowed[name] = struct{}{}
	}

	out := make([]spec.ToolSpec, 0, len(tools))
	for _, t := range tools {
		if _, ok := allowed[t.Name()]; ok {
			out = append(out, t)
		}
	}

	return out
}
