package background

import (
	"github.com/vitaliiPsl/crappy-adk/agent"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
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

func New(manager *bg.Manager) extension.Extension {
	return &ext{manager: manager}
}

func (e *ext) Name() string {
	return "background"
}

func (e *ext) Options(extension.Context) (agent.Option, error) {
	return agent.WithExtensions(
		agent.WithInstructions(Instructions),
		agent.WithTools(bg.Tools(e.manager)...),
	), nil
}
