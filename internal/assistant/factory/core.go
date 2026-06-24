package factory

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/guard"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/instructions"
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
	handler     permission.Handler
}

func (e coreContributor) Name() string {
	return coreSource
}

func (e coreContributor) Options(_ context.Context, req BuildRequest) ([]kit.Tool, []agent.Option, error) {
	options := e.options(req)

	return e.tools(req), options, nil
}

func (e coreContributor) tools(req BuildRequest) []kit.Tool {
	return tools.Core(e.background.ForSession(req.SessionID))
}

func (e coreContributor) options(req BuildRequest) []agent.Option {
	instructions := agent.WithInstructions(
		req.Config.Prompt,
		instructions.Env(req.Config.Cwd),
		instructions.Files(req.Config.Cwd),
	)

	options := []agent.Option{instructions}

	options = append(options, e.generationOptions(req)...)

	options = append(options,
		e.permissionHook(req),
		guard.WithRepeatedToolCallLimit(toolLoopMaxRepeats, toolLoopWindow),
	)

	return options
}

func (e coreContributor) generationOptions(req BuildRequest) []agent.Option {
	var options []agent.Option

	if req.Config.Thinking != "" {
		options = append(options, agent.WithThinking(kit.ThinkingLevel(req.Config.Thinking)))
	}

	if req.Config.Temperature != nil {
		options = append(options, agent.WithTemperature(*req.Config.Temperature))
	}

	if req.Config.MaxOutputTokens != nil {
		options = append(options, agent.WithMaxOutputTokens(*req.Config.MaxOutputTokens))
	}

	return options
}

func (e coreContributor) permissionHook(req BuildRequest) agent.Option {
	auth := permission.Context{
		SessionID:   req.SessionID,
		Mode:        req.Config.Mode,
		Permissions: req.Config.Permissions,
		Handler:     e.handler,
	}

	return agent.WithOnToolCall(func(rc *kit.RunContext, call kit.ToolCall) (kit.ToolCall, error) {
		if err := e.permissions.Authorize(rc.Context, auth, call); err != nil {
			return call, err
		}

		return call, nil
	})
}
