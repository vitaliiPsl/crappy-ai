# Configuration

Configure Crappy with two YAML files: config and settings.

Config controls how the assistant behaves while working. Settings control how Crappy itself is wired up on your machine.

## Config vs Settings

Use config for things you may change often while working:

- system prompt
- active provider
- active model
- thinking level
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
system_prompt: |
  You are Crappy, an AI assistant for work on the user's machine.
  ...

provider: anthropic
model: claude-sonnet-4-6
thinking: medium
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

`system_prompt` changes the assistant's behavior.

`mode` controls the assistant's permission behavior. Use `default` for normal permission rules or `yolo` to allow all tool calls without prompting.

`permissions` control which tools can run automatically, which ones ask first, and which ones are blocked.

## Settings File

The settings file describes providers, storage paths, and model metadata.

```yaml
config_path: ~/.crappy-ai/config.yaml
sessions_dir: ~/.crappy-ai/sessions
models_path: ~/.crappy-ai/models.json
skills_path: ~/.crappy-ai/skills

providers:
  - name: openai
    api: openai
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

`providers` tells Crappy how to connect to model providers.

`models` adds or overrides model metadata for a provider.

Provider API keys are usually read from environment variables:

```sh
export OPENAI_API_KEY=...
```

You can also set `api_key` directly in settings, but environment variables are safer for real credentials.

## Provider Settings

A provider entry has a `name` and an `api`.

```yaml
providers:
  - name: openai-local
    api: openai
    base_url: http://localhost:11434/v1
    api_key: local
```

`name` is what you select in config.

`api` tells Crappy which provider protocol to use.

`base_url` points Crappy at a compatible gateway, proxy, or local model runtime.

See [Models and Providers](models.md) for provider APIs, custom providers, and custom model metadata.

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

Provider API keys use the environment variable named by `api_key_env`, such as `OPENAI_API_KEY`.

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
