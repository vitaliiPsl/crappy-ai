package assistant

import (
	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/guard"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/instructions"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/planning"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/summarization"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
)

const (
	toolLoopMaxRepeats = 3
	toolLoopWindow     = 5
)

func (a *Assistant) buildAgentOpts(sessionID string, cfg config.Config, model kit.Model) []agent.Option {
	opts := []agent.Option{
		agent.WithInstructions(
			cfg.SystemPrompt,
			instructions.Env(cfg.Cwd),
			instructions.Files(cfg.Cwd),
		),
		agent.WithTools(a.toolRegistry.GetTools()...),
		guard.WithRepeatedToolCallLimit(toolLoopMaxRepeats, toolLoopWindow),
		summarization.New(model),
		planning.New(sessionID, a.artifactStore),
	}

	if cfg.Thinking != "" {
		opts = append(opts, agent.WithThinking(kit.ThinkingLevel(cfg.Thinking)))
	}

	opts = append(opts, agent.WithOnToolCall(func(rc *kit.RunContext, call kit.ToolCall) (kit.ToolCall, error) {
		if err := a.permissions.Authorize(rc.Context, sessionID, call); err != nil {
			return call, err
		}

		return call, nil
	}))

	return opts
}
