package runtime

import (
	"context"
	"fmt"
	"strings"

	adk "github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/tool"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

const (
	subagentToolName        = "task"
	subagentToolDescription = "Delegate a task to a specialized subagent. Returns the subagent's output, or set background to run it as a job and collect the result with job_result."
)

const subagentInstructions = `# Subagents

You have a task tool that delegates work to a specialized subagent.

## When to use it
- A task matches a subagent's stated purpose.
- You want to isolate exploratory or noisy work from the main conversation.

## How to use it
- Call task with the subagent name and a self-contained description of the work.
- By default the subagent runs to completion and returns its output directly.
- For long-running work, set "background": true to get a job_id and collect the result later with job_result.

Available subagents:
`

type SubagentRequest struct {
	Agent       string
	Task        string
	Description string
}

type SubagentResult struct {
	SessionID string
	Output    string
	Usage     kit.Usage
	LastUsage kit.Usage
}

type subagentsContributor struct {
	session    *Session
	background *background.Manager
}

type subagentToolInput struct {
	Agent       string `json:"agent" jsonschema:"Name of the subagent to run, from the available subagents list"`
	Task        string `json:"task" jsonschema:"A self-contained description of the work, including all context the subagent needs"`
	Description string `json:"description" jsonschema:"A short (3-5 word) description of the task, used to label the subagent's session"`
}

func (c subagentsContributor) Contribute(_ context.Context, req appagent.Request) (appagent.Contribution, error) {
	if c.session == nil || len(req.Config.Agents) == 0 {
		return appagent.Contribution{}, nil
	}

	var taskTool kit.Tool = tool.MustNew(
		subagentToolName,
		subagentToolDescription,
		func(rc *kit.RunContext, input subagentToolInput) (string, error) {
			result, err := c.session.runSubagent(rc.Context, SubagentRequest(input))
			if err != nil {
				return "", err
			}

			return result.Output, nil
		},
	)

	if c.background != nil {
		wrapped, err := background.Wrap(taskTool, c.background.ForSession(req.Session.ID))
		if err != nil {
			return appagent.Contribution{}, err
		}

		taskTool = wrapped
	}

	return appagent.Contribution{
		Tools: []kit.Tool{taskTool},
		Options: []adk.Option{
			adk.WithInstructions(subagentInstructions + subagentListing(req.Config.Agents)),
		},
	}, nil
}

func (s *Session) runSubagent(ctx context.Context, req SubagentRequest) (SubagentResult, error) {
	cfg := s.configStore.Get()

	sub, ok := cfg.Subagent(req.Agent)
	if !ok {
		return SubagentResult{}, fmt.Errorf("unknown subagent %q", req.Agent)
	}

	parent, err := s.sessionStore.Get(ctx, s.id)
	if err != nil {
		return SubagentResult{}, fmt.Errorf("load parent session: %w", err)
	}

	model, err := s.modelRegistry.Build(ctx, sub.Provider, sub.Model)
	if err != nil {
		return SubagentResult{}, fmt.Errorf("build subagent %q model: %w", req.Agent, err)
	}

	child, err := s.sessionStore.Create(ctx, session.CreateParams{
		Title:    subagentTitle(req),
		Cwd:      parent.Cwd,
		ParentID: s.id,
	})
	if err != nil {
		return SubagentResult{}, fmt.Errorf("create subagent session: %w", err)
	}

	childCfg := cfg
	childCfg.Agent = sub
	childCfg.Agents = nil

	ag, err := s.buildAgent(ctx, *child, childCfg, model, newMemory(s.sessionStore, child.ID))
	if err != nil {
		return SubagentResult{}, err
	}

	resp, err := ag.Run(ctx, kit.NewUserMessage(kit.NewTextContent(req.Task)))
	if err != nil {
		return SubagentResult{}, err
	}

	s.recordUsage(ctx, child.ID, resp.Usage)
	s.recordUsage(ctx, s.id, resp.Usage)

	result := SubagentResult{
		SessionID: child.ID,
		Usage:     resp.Usage,
		LastUsage: resp.LastUsage,
	}
	if resp.Output != nil {
		result.Output = resp.Output.Text
	}

	return result, nil
}

func subagentTitle(req SubagentRequest) string {
	if req.Description == "" {
		return req.Agent
	}

	return req.Agent + ": " + req.Description
}

func subagentListing(agents []config.Agent) string {
	var b strings.Builder
	for _, agent := range agents {
		b.WriteString("- ")
		b.WriteString(agent.Name)

		if agent.Description != "" {
			b.WriteString(": ")
			b.WriteString(agent.Description)
		}

		b.WriteString("\n")
	}

	return b.String()
}
