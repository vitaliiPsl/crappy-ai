package assistant

import (
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/guard"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/instructions"
)

const (
	toolLoopMaxRepeats = 3
	toolLoopWindow     = 5
)

func (a *Assistant) buildAgentOpts(ctx extension.Context) ([]agent.Option, error) {
	cfg := ctx.Config

	opts := []agent.Option{
		agent.WithInstructions(
			cfg.SystemPrompt,
			instructions.Env(cfg.Cwd),
			instructions.Files(cfg.Cwd),
		),
		guard.WithRepeatedToolCallLimit(toolLoopMaxRepeats, toolLoopWindow),
	}

	if cfg.Thinking != "" {
		opts = append(opts, agent.WithThinking(kit.ThinkingLevel(cfg.Thinking)))
	}

	opts = append(opts, agent.WithOnToolCall(func(rc *kit.RunContext, call kit.ToolCall) (kit.ToolCall, error) {
		if err := a.permissions.Authorize(rc.Context, ctx.SessionID, call); err != nil {
			return call, err
		}

		return call, nil
	}))

	for _, ext := range a.extensions {
		opt, err := ext.Options(ctx)
		if err != nil {
			return nil, fmt.Errorf("extension %q: %w", ext.Name(), err)
		}

		opts = append(opts, opt)
	}

	return opts, nil
}
