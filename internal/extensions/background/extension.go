package background

import (
	"context"

	adk "github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
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

type Extension interface {
	factory.Extension
	appagent.Contributor
}

var _ factory.Extension = (*ext)(nil)
var _ appagent.Contributor = (*ext)(nil)

func New(manager *bg.Manager) Extension {
	return &ext{manager: manager}
}

func (e *ext) Name() string {
	return "background"
}

func (e *ext) Options(_ context.Context, req factory.BuildRequest) ([]kit.Tool, []adk.Option, error) {
	c := e.contribution(req.SessionID)

	return c.Tools, c.Options, nil
}

func (e *ext) Contribute(_ context.Context, req appagent.Request) (appagent.Contribution, error) {
	return e.contribution(req.SessionID), nil
}

func (e *ext) contribution(sessionID string) appagent.Contribution {
	if e.manager == nil {
		return appagent.Contribution{}
	}

	jobs := e.manager.ForSession(sessionID)

	return appagent.Contribution{
		Tools: bg.Tools(jobs),
		Options: []adk.Option{
			adk.WithInstructions(Instructions),
		},
	}
}
