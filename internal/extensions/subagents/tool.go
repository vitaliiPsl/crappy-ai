package subagents

import (
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/memory"
	"github.com/vitaliiPsl/crappy-adk/x/tool"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
)

const (
	toolName        = "task"
	toolDescription = "Delegate a task to a specialized subagent. Returns the subagent's output, or set background to run it as a job and collect the result with job_result."
)

type taskInput struct {
	Agent string `json:"agent" jsonschema:"Name of the subagent to run, from the available subagents list"`
	Task  string `json:"task" jsonschema:"A self-contained description of the work, including all context the subagent needs"`
}

func newTool(f *factory.Factory, extensions []factory.Extension, registry *models.Registry, ec factory.Context) kit.Tool {
	return tool.MustNew(
		toolName,
		toolDescription,
		func(rc *kit.RunContext, input taskInput) (string, error) {
			sub, ok := ec.Config.Subagent(input.Agent)
			if !ok {
				return "", fmt.Errorf("unknown subagent %q", input.Agent)
			}

			model, err := registry.Build(sub.Provider, sub.Model)
			if err != nil {
				return "", fmt.Errorf("build subagent %q model: %w", input.Agent, err)
			}

			childCfg := ec.Config
			childCfg.Agent = sub

			ag, err := f.Build(factory.BuildRequest{
				Context: factory.Context{
					Ctx:       rc.Context,
					SessionID: ec.SessionID + "/sub:" + input.Agent,
					Config:    childCfg,
					Model:     model,
				},
				Memory:     memory.NewHistory(),
				Extensions: extensions,
			})
			if err != nil {
				return "", err
			}

			resp, err := ag.Run(rc.Context, kit.NewUserMessage(kit.NewTextContent(input.Task)))
			if err != nil {
				return "", err
			}

			if resp.Output == nil {
				return "", nil
			}

			return resp.Output.Text, nil
		},
	)
}
