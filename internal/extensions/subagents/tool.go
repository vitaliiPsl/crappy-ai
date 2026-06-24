package subagents

import (
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/tool"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/memory"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

const (
	toolName        = "task"
	toolDescription = "Delegate a task to a specialized subagent. Returns the subagent's output, or set background to run it as a job and collect the result with job_result."
)

type taskInput struct {
	Agent       string `json:"agent" jsonschema:"Name of the subagent to run, from the available subagents list"`
	Task        string `json:"task" jsonschema:"A self-contained description of the work, including all context the subagent needs"`
	Description string `json:"description" jsonschema:"A short (3-5 word) description of the task, used to label the subagent's session"`
}

func (e *ext) newTool(req factory.BuildRequest) kit.Tool {
	return tool.MustNew(
		toolName,
		toolDescription,
		func(rc *kit.RunContext, input taskInput) (string, error) {
			return e.runSubagent(rc, req, input)
		},
	)
}

func (e *ext) runSubagent(rc *kit.RunContext, req factory.BuildRequest, input taskInput) (string, error) {
	sub, ok := req.Config.Subagent(input.Agent)
	if !ok {
		return "", fmt.Errorf("unknown subagent %q", input.Agent)
	}

	model, err := e.modelRegistry.Build(sub.Provider, sub.Model)
	if err != nil {
		return "", fmt.Errorf("build subagent %q model: %w", input.Agent, err)
	}

	child, err := e.sessionStore.Create(rc.Context, session.CreateParams{
		Title:    subagentTitle(input),
		Cwd:      req.Config.Cwd,
		ParentID: req.SessionID,
	})
	if err != nil {
		return "", fmt.Errorf("create subagent session: %w", err)
	}

	childCfg := req.Config
	childCfg.Agent = sub

	ag, err := e.factory.Build(rc.Context, factory.BuildRequest{
		SessionID:  child.ID,
		Config:     childCfg,
		Model:      model,
		Memory:     memory.New(e.sessionStore, child.ID),
		Extensions: e.extensions,
	})
	if err != nil {
		return "", err
	}

	resp, err := ag.Run(rc.Context, kit.NewUserMessage(kit.NewTextContent(input.Task)))
	if err != nil {
		return "", err
	}

	e.recordUsage(rc, child.ID, req.SessionID, resp.Usage)

	if resp.Output == nil {
		return "", nil
	}

	return resp.Output.Text, nil
}

func (e *ext) recordUsage(rc *kit.RunContext, childID, parentID string, usage kit.Usage) {
	for _, id := range []string{childID, parentID} {
		sess, err := e.sessionStore.Get(rc.Context, id)
		if err != nil {
			continue
		}

		sess.Usage.Add(usage)
		_ = e.sessionStore.Save(rc.Context, sess)
	}
}

func subagentTitle(input taskInput) string {
	if input.Description == "" {
		return input.Agent
	}

	return input.Agent + ": " + input.Description
}
