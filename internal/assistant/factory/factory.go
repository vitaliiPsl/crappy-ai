package factory

import (
	"context"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
)

type Context struct {
	Ctx       context.Context
	SessionID string
	Config    config.Config
	Model     kit.Model
}

type BuildRequest struct {
	Context

	Memory     kit.Memory
	Extensions []Extension
}

type Extension interface {
	Name() string
	Spec(ctx Context) (AgentSpec, error)
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

func (f *Factory) Build(req BuildRequest) (*agent.Agent, error) {
	runSpec, err := f.buildSpec(req)
	if err != nil {
		return nil, fmt.Errorf("build agent spec: %w", err)
	}

	compiled, err := Compile(runSpec)
	if err != nil {
		return nil, fmt.Errorf("compile agent spec: %w", err)
	}

	if req.Config.Thinking != "" {
		compiled.Options = append(compiled.Options, agent.WithThinking(kit.ThinkingLevel(req.Config.Thinking)))
	}

	ag, err := agent.New(req.Model, req.Memory, compiled.Tools, compiled.Options...)
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	return ag, nil
}

func (f *Factory) buildSpec(req BuildRequest) (AgentSpec, error) {
	runSpec, err := collectSpecs(req.Context, f.contributors(req.Extensions))
	if err != nil {
		return AgentSpec{}, err
	}

	runSpec.Tools = allowedTools(runSpec.Tools, req.Config.Tools)

	return runSpec, nil
}

func (f *Factory) contributors(extensions []Extension) []Extension {
	out := make([]Extension, 0, len(extensions)+1)
	out = append(out, coreContributor{
		permissions: f.permissions,
		background:  f.background,
	})
	out = append(out, extensions...)

	return out
}

func collectSpecs(ctx Context, contributors []Extension) (AgentSpec, error) {
	var runSpec AgentSpec
	for _, contributor := range contributors {
		contributed, err := contributor.Spec(ctx)
		if err != nil {
			return AgentSpec{}, fmt.Errorf("%s spec: %w", contributor.Name(), err)
		}

		runSpec.Merge(contributed)
	}

	return runSpec, nil
}

func allowedTools(tools []ToolSpec, allow []string) []ToolSpec {
	if len(allow) == 0 {
		return tools
	}

	allowed := make(map[string]struct{}, len(allow))
	for _, name := range allow {
		allowed[name] = struct{}{}
	}

	out := make([]ToolSpec, 0, len(tools))
	for _, t := range tools {
		if _, ok := allowed[t.Name()]; ok {
			out = append(out, t)
		}
	}

	return out
}
