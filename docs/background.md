# Background Execution

Background execution lets Crappy start long-running tool work without blocking the conversation.

When Crappy runs `bash` with `background: true`, the command starts as a job and the tool returns immediately with a `job_id`.

## When Crappy Uses It

Crappy uses background execution for shell commands that may keep running while it continues other work:

- dev servers
- file watchers
- slow builds
- long test suites
- commands that produce output later

Crappy should use normal foreground `bash` calls for quick commands where it needs the output immediately.

## Jobs

When Crappy starts a background job, it calls a background-capable tool with `background: true`.

For `bash`, the tool arguments look like:

```json
{
  "command": "npm run dev",
  "description": "Start the dev server",
  "background": true
}
```

The tool result is a job snapshot:

```json
{
  "job_id": "job_1",
  "tool": "bash",
  "status": "running",
  "started_at": "2026-06-06T08:00:00Z"
}
```

The command keeps running after the assistant turn continues. The job is owned by the current Crappy process, not by the single assistant turn.

## Job Tools

Crappy has generic job tools for inspecting and canceling background work.

### `job_status`

Get the current status of a background job.

```json
{
  "job_id": "job_1"
}
```

Statuses are:

- `running`
- `succeeded`
- `failed`
- `canceled`

### `job_result`

Get the result of a background job.

If the job is still running, this returns the current job status. When the job has finished, it includes the final output or error.

### `job_list`

List background jobs, newest first. Crappy can use this if it needs to recover a job ID.

### `job_cancel`

Cancel a running background job.

```json
{
  "job_id": "job_1"
}
```

Canceling sends cancellation to the running tool. For `bash`, that cancels the shell command through the process context.

## Permissions

Starting a background `bash` job uses the normal `bash` permission rules. The `background` argument does not bypass approval.

For example, this rule allows both foreground and background `go test` commands:

```yaml
permissions:
  allow:
    - tool: bash
      pattern: "go test *"
```

Job-control tools are allowed by default:

- `job_status`
- `job_result`
- `job_list`
- `job_cancel`

You can deny a job-control tool in config if needed:

```yaml
permissions:
  deny:
    - tool: job_cancel
```

## Lifetime

Background jobs live in memory for the current Crappy process.

They are not restored after Crappy exits, and their output is not currently streamed into the session event log. Crappy can inspect them with `job_status`, `job_result`, and `job_list` while the process is running.

When Crappy shuts down, running jobs are canceled.

## Notes

Background execution is an execution mode for tools, not a separate shell tool. `bash` is still the tool that runs commands; `background: true` changes how that tool call is scheduled.

Future long-running tools, such as subagents, can use the same job-control tools.
