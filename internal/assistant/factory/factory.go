package factory

import (
	"context"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	xtool "github.com/vitaliiPsl/crappy-adk/x/tool"

	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
)

type Extension interface {
	Name() string
	Options(ctx context.Context, req BuildRequest) ([]kit.Tool, []agent.Option, error)
}

type BuildRequest struct {
	SessionID string
	Config    config.Config

	Model  kit.Model
	Memory kit.Memory

	Extensions        []Extension
	PermissionHandler permission.Handler
}

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

func (f *Factory) Build(ctx context.Context, req BuildRequest) (*agent.Agent, error) {
	tools, options, err := f.configure(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("collect agent options: %w", err)
	}

	ag, err := agent.New(req.Model, req.Memory, xtool.NewSet(tools...), options...)
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	return ag, nil
}

func (f *Factory) configure(ctx context.Context, req BuildRequest) ([]kit.Tool, []agent.Option, error) {
	tools, options, err := collect(ctx, req, f.contributors(req))
	if err != nil {
		return nil, nil, err
	}

	tools = allowedTools(tools, req.Config.Tools)

	return tools, options, nil
}

func (f *Factory) contributors(req BuildRequest) []Extension {
	out := make([]Extension, 0, len(req.Extensions)+1)
	out = append(out, coreContributor{
		permissions: f.permissions,
		background:  f.background,
		handler:     req.PermissionHandler,
	})
	out = append(out, req.Extensions...)

	return out
}

func collect(ctx context.Context, req BuildRequest, contributors []Extension) ([]kit.Tool, []agent.Option, error) {
	var (
		tools   []kit.Tool
		options []agent.Option
	)

	for _, contributor := range contributors {
		contributedTools, contributedOptions, err := contributor.Options(ctx, req)
		if err != nil {
			return nil, nil, fmt.Errorf("%s options: %w", contributor.Name(), err)
		}

		tools = append(tools, contributedTools...)
		options = append(options, contributedOptions...)
	}

	return tools, options, nil
}

func allowedTools(tools []kit.Tool, allow []string) []kit.Tool {
	if len(allow) == 0 {
		return tools
	}

	allowed := make(map[string]struct{}, len(allow))
	for _, name := range allow {
		allowed[name] = struct{}{}
	}

	out := make([]kit.Tool, 0, len(tools))
	for _, t := range tools {
		if _, ok := allowed[t.Definition().Name]; ok {
			out = append(out, t)
		}
	}

	return out
}
