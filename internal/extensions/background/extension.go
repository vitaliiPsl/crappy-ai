package background

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	bg "github.com/vitaliiPsl/crappy-ai/internal/background"
)

const Instructions = `# Background Jobs

Some tools support a background execution.

Use background jobs for long-running commands such as dev servers, watchers, builds, and slow test suites when waiting would block the conversation.

When you start a background job:
- Set "background": true on the original tool call.
- Use job_status, job_result, job_list, and job_cancel to inspect or stop jobs.
- Do not assume a background job has finished until you check it.

Use foreground tool calls for quick commands where the immediate output is needed.`

type ext struct {
	manager *bg.Manager
}

func New(manager *bg.Manager) factory.Extension {
	return &ext{manager: manager}
}

func (e *ext) Name() string {
	return "background"
}

func (e *ext) Options(_ context.Context, req factory.BuildRequest) ([]kit.Tool, []agent.Option, error) {
	jobs := e.manager.ForSession(req.SessionID)

	return bg.Tools(jobs), []agent.Option{
		agent.WithInstructions(Instructions),
	}, nil
}
