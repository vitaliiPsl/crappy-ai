package background

import (
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
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

func (e *ext) Context(extension.Context) ([]spec.ContextPiece, error) {
	return []spec.ContextPiece{
		{
			Name:    "Background job instructions",
			Source:  e.Name(),
			Kind:    spec.ContextExtension,
			Content: Instructions,
		},
	}, nil
}

func (e *ext) Tools(ctx extension.Context) ([]spec.ToolSpec, error) {
	jobs := e.manager.ForSession(ctx.SessionID)
	jobTools := bg.Tools(jobs)

	tools := make([]spec.ToolSpec, 0, len(jobTools))
	for _, t := range jobTools {
		tools = append(tools, spec.ToolSpec{
			Source: e.Name(),
			Tool:   t,
		})
	}

	return tools, nil
}

func (e *ext) Hooks(extension.Context) ([]spec.HookSpec, error) {
	return nil, nil
}
