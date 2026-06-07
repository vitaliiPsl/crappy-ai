package background

import (
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/tool"
)

type jobArgs struct {
	ID string `json:"job_id" jsonschema:"The background job ID."`
}

type listArgs struct{}

func Tools(jobs Jobs) []kit.Tool {
	return []kit.Tool{
		statusTool(jobs),
		resultTool(jobs),
		cancelTool(jobs),
		listTool(jobs),
	}
}

func statusTool(jobs Jobs) kit.Tool {
	return tool.MustNew(
		ToolStatus,
		"Get the current status of a background job.",
		func(_ *kit.RunContext, input jobArgs) (Job, error) {
			return jobs.Get(input.ID)
		},
	)
}

func resultTool(jobs Jobs) kit.Tool {
	return tool.MustNew(
		ToolResult,
		"Get the result of a background job. Running jobs return their current status without output.",
		func(_ *kit.RunContext, input jobArgs) (Job, error) {
			return jobs.Get(input.ID)
		},
	)
}

func cancelTool(jobs Jobs) kit.Tool {
	return tool.MustNew(
		ToolCancel,
		"Cancel a running background job.",
		func(_ *kit.RunContext, input jobArgs) (Job, error) {
			return jobs.Cancel(input.ID)
		},
	)
}

func listTool(jobs Jobs) kit.Tool {
	return tool.MustNew(
		ToolList,
		"List background jobs, newest first.",
		func(_ *kit.RunContext, _ listArgs) ([]Job, error) {
			return jobs.List()
		},
	)
}
