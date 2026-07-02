package agent

import (
	"context"

	adk "github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	xtool "github.com/vitaliiPsl/crappy-adk/x/tool"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
)

type Request struct {
	SessionID string
	Config    config.Config

	Model  kit.Model
	Memory kit.Memory
	Asker  ask.Asker
}

type Contribution struct {
	Tools   []kit.Tool
	Options []adk.Option
}

type Contributor interface {
	Contribute(context.Context, Request) (Contribution, error)
}

func Build(ctx context.Context, req Request, contributors ...Contributor) (*adk.Agent, error) {
	var contribution Contribution
	for _, contributor := range contributors {
		c, err := contributor.Contribute(ctx, req)
		if err != nil {
			return nil, err
		}

		contribution.Tools = append(contribution.Tools, c.Tools...)
		contribution.Options = append(contribution.Options, c.Options...)
	}

	contribution.Tools = filterTools(contribution.Tools, req.Config.Tools)

	return adk.New(req.Model, req.Memory, xtool.NewSet(contribution.Tools...), contribution.Options...)
}

func filterTools(tools []kit.Tool, allow []string) []kit.Tool {
	if len(allow) == 0 {
		return tools
	}

	allowed := make(map[string]struct{}, len(allow))
	for _, name := range allow {
		allowed[name] = struct{}{}
	}

	out := make([]kit.Tool, 0, len(tools))
	for _, tool := range tools {
		if _, ok := allowed[tool.Definition().Name]; ok {
			out = append(out, tool)
		}
	}

	return out
}
