# Configuration

Configure Crappy with two YAML files: config and settings.

Config controls how the assistant behaves while working. Settings control how Crappy itself is wired up on your machine.

## Config vs Settings

Use config for things you may change often while working:

- system prompt
- active provider
- active model
- thinking level
- temperature and output limit
- mode
- permissions

Use settings for things that describe your local installation:

- where Crappy stores files
- provider credentials
- provider endpoints
- custom model metadata
- user skills directory

By default, Crappy uses:

```text
~/.crappy-ai/config.yaml
~/.crappy-ai/settings.yaml
```

Crappy creates these files automatically the first time it runs.

## Config File

The config file is the main place to tune assistant behavior.

```yaml
prompt: |
  You are Crappy, an AI assistant for work on the user's machine.
  ...

provider: anthropic
model: claude-sonnet-4-6
thinking: medium
temperature: 0.2
max_output_tokens: 4096
mode: default

permissions:
  default: ask
  allow:
    - tool: list
      pattern: "./**"
    - tool: read_file
      pattern: "./**"
```

`provider` and `model` choose the model Crappy uses.

`thinking` sets the reasoning level passed to the model:

- `disabled`
- `low`
- `medium`
- `high`

`temperature` controls sampling randomness. Lower values are more deterministic; higher values are more varied.

`max_output_tokens` limits the number of tokens the model can generate in one model call.

`prompt` changes the assistant's behavior.

`mode` controls the assistant's permission behavior. Use `default` for normal permission rules or `yolo` to allow all tool calls without prompting.

## Subagents

`agents` defines named subagents the assistant can delegate to with the `task` tool. Each entry reuses the same fields as the root agent, plus `name`, `description`, and a `tools` allowlist.

```yaml
agents:
  - name: explorer
    description: Read-only search agent for broad codebase exploration.
    prompt: |
      You are a read-only exploration agent. Search and summarize; do not edit.
    tools: [read_file, list, bash]
    permissions:
      default: deny
      allow:
        - tool: read_file
          pattern: ./...
        - tool: list
          pattern: ./...
```

- `name` and `description` identify the subagent in the available-subagents list.
- `provider`, `model`, `thinking`, `temperature`, and `max_output_tokens` inherit from the root agent when omitted; everything else (`prompt`, `permissions`, `tools`) is the subagent's own.
- `tools` is an allowlist: the subagent only sees those tools.
- `permissions` are evaluated for the subagent's own tool calls. If a subagent tool call needs approval, Crappy prompts in the parent session.

By default, `task` runs the subagent to completion and returns its final output. The subagent gets a persistent child session with isolated session memory and `parent_id` set to the calling session.

For longer work, `task` also supports background execution:

```json
{
  "agent": "explorer",
  "description": "trace auth flow",
  "task": "Find where authentication is implemented and summarize the flow.",
  "background": true
}
```

With `background: true`, `task` returns a `job_id`; use `job_status`, `job_result`, `job_list`, and `job_cancel` to inspect or stop the job.

`permissions` control which tools can run automatically, which ones ask first, and which ones are blocked.

## Settings File

The settings file describes providers, storage paths, and model metadata.

```yaml
config_path: ~/.crappy-ai/config.yaml
sessions_dir: ~/.crappy-ai/sessions
models_path: ~/.crappy-ai/models.json
skills_path: ~/.crappy-ai/skills
memory_path: ~/.crappy-ai/memory.json

providers:
  - id: openai
    api: openai
    auth:
      type: api_key
      api_key_env: OPENAI_API_KEY

models:
  openai-local:
    - id: gemma4
      context_window: 131072
```

`config_path` points to the config file.

`sessions_dir` is where Crappy stores sessions.

`models_path` is where Crappy stores model metadata.

`skills_path` is where Crappy loads user-level skills.

`memory_path` is where Crappy stores explicitly saved persistent memories.

`providers` tells Crappy how to connect to model providers.

`models` adds or overrides model metadata for a provider.

Provider API keys are usually read from environment variables:

```sh
export OPENAI_API_KEY=...
```

You can also set `auth.api_key` directly in settings, but environment variables are safer for real credentials.

## Provider Settings

A provider entry has an `id` and an `api`.

```yaml
providers:
  - id: openai-local
    api: openai
    base_url: http://localhost:11434/v1
    auth:
      type: api_key
      api_key: local
```

`id` is what you select in config.

`api` tells Crappy which provider protocol to use.

`base_url` points Crappy at a compatible gateway, proxy, or local model runtime.

`auth` selects API-key or OAuth authentication.

See [Models and Providers](models.md) for provider APIs, custom providers, authentication, and custom model metadata.

## Permissions

Permissions live in config because they affect what the assistant can do while working.

```yaml
mode: default
permissions:
  default: ask
  allow:
    - tool: list
      pattern: "./**"
    - tool: read_file
      pattern: "./**"
  deny:
    - tool: bash
      pattern: "rm *"
```

Set `mode` to `yolo` to allow all tool calls without prompting.

See [Permissions](permissions.md) for modes, rule syntax, and examples.

## Environment Variables

Environment variables can override active model and permission settings:

```sh
CRAPPY_PROVIDER=openai
CRAPPY_MODEL=gpt-5.5
CRAPPY_THINKING=high
CRAPPY_MODE=yolo
```

They can also change where Crappy stores settings and session data:

```sh
CRAPPY_SETTINGS=~/.config/crappy/settings.yaml
CRAPPY_SESSIONS_DIR=~/.local/share/crappy/sessions
CRAPPY_MODELS_PATH=~/.cache/crappy/models.json
CRAPPY_SKILLS_PATH=~/.config/crappy/skills
```

Provider API keys use the environment variable named by `auth.api_key_env`, such as `OPENAI_API_KEY`.

## CLI Overrides

Override model and permission settings for a single run with CLI flags:

```sh
crappy -provider openai -model gpt-5.5 -thinking high
crappy -mode yolo
```

CLI flags override environment variables and config values.

## Precedence

Config values are applied in this order:

1. defaults
2. config file
3. environment variables
4. CLI flags

Later values override earlier values.

Settings values are applied in this order:

1. defaults
2. settings file
3. environment variables

Provider entries in settings are merged by name. Model metadata is merged by provider and model ID.

That means you can override one provider, add another provider, or add custom model metadata without replacing everything else.

Permissions are also merged instead of replaced.

## Working Directory

Crappy uses the directory where it was started as the current workspace.

Relative permission patterns such as `./**` are resolved from that directory. Sessions also remember the working directory they were created in.
