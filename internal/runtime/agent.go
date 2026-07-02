package runtime

import (
	"context"

	adk "github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/guard"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
	"github.com/vitaliiPsl/crappy-ai/internal/agent/instructions"
	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/tools/bash"
	filesystem "github.com/vitaliiPsl/crappy-ai/internal/tools/fs"
	"github.com/vitaliiPsl/crappy-ai/internal/tools/web"
)

const (
	toolLoopMaxRepeats = 3
	toolLoopWindow     = 5
)

type coreContributor struct {
	background *background.Manager
}

func (c coreContributor) Contribute(_ context.Context, req appagent.Request) (appagent.Contribution, error) {
	tools, err := c.tools(req)
	if err != nil {
		return appagent.Contribution{}, err
	}

	return appagent.Contribution{
		Tools:   tools,
		Options: coreOptions(req),
	}, nil
}

func coreOptions(req appagent.Request) []adk.Option {
	var opts []adk.Option

	opts = append(opts, instructionOptions(req)...)
	opts = append(opts, generationOptions(req)...)
	opts = append(opts, guardOptions()...)

	return opts
}

func instructionOptions(req appagent.Request) []adk.Option {
	return []adk.Option{
		adk.WithInstructions(
			req.Config.Prompt,
			instructions.Env(req.Config.Cwd),
			instructions.Files(req.Config.Cwd),
		),
	}
}

func generationOptions(req appagent.Request) []adk.Option {
	var opts []adk.Option

	if req.Config.Thinking != "" {
		opts = append(opts, adk.WithThinking(kit.ThinkingLevel(req.Config.Thinking)))
	}

	if req.Config.Temperature != nil {
		opts = append(opts, adk.WithTemperature(*req.Config.Temperature))
	}

	if req.Config.MaxOutputTokens != nil {
		opts = append(opts, adk.WithMaxOutputTokens(*req.Config.MaxOutputTokens))
	}

	return opts
}

func guardOptions() []adk.Option {
	return []adk.Option{
		guard.WithRepeatedToolCallLimit(toolLoopMaxRepeats, toolLoopWindow),
	}
}

func (c coreContributor) tools(req appagent.Request) ([]kit.Tool, error) {
	bashTool := bash.NewBash()
	if c.background != nil {
		wrapped, err := background.Wrap(bashTool, c.background.ForSession(req.SessionID))
		if err != nil {
			return nil, err
		}

		bashTool = wrapped
	}

	return []kit.Tool{
		bashTool,
		web.NewFetch(),
		filesystem.NewReadFile(),
		filesystem.NewWriteFile(),
		filesystem.NewEditFile(),
		filesystem.NewListDirectory(),
	}, nil
}
