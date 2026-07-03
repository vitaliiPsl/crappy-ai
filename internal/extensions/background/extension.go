package background

import (
	"context"

	adk "github.com/vitaliiPsl/crappy-adk/agent"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
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

var _ appagent.Contributor = (*ext)(nil)

func New(manager *bg.Manager) appagent.Contributor {
	return &ext{
		manager: manager,
	}
}

func (e *ext) Contribute(_ context.Context, req appagent.Request) (appagent.Contribution, error) {
	jobs := e.manager.ForSession(req.SessionID)

	return appagent.Contribution{
		Tools: bg.Tools(jobs),
		Options: []adk.Option{
			adk.WithInstructions(Instructions),
		},
	}, nil
}
