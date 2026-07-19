# Models and Providers

Choose which model Crappy uses and how it connects to model providers.

Crappy keeps model choice and provider credentials separate. The active provider and model live in config. Provider credentials, base URLs, and custom model metadata live in settings.

## Quick Config

Set the active provider and model in `~/.crappy-ai/config.yaml`.

```yaml
provider: anthropic
model: claude-sonnet-4-6
```

Override them for a single run with CLI flags:

```sh
crappy -provider openai -model gpt-5.5
```

Or with environment variables:

```sh
CRAPPY_PROVIDER=google
CRAPPY_MODEL=gemini-3.5-flash
```

CLI flags override environment variables. Environment variables override the config file.

## Provider APIs

Crappy can speak these provider APIs:

- `anthropic` uses Anthropic's Messages API for Claude models.
- `openai` uses OpenAI's Responses API. Use this for OpenAI models and OpenAI-compatible servers.
- `google` uses Google's Gemini API for Gemini models.

A provider entry maps a name to one of those APIs.

```yaml
providers:
  - id: openai
    api: openai
    auth:
      type: api_key
      api_key_env: OPENAI_API_KEY
```

`id` is what you select in config. `api` tells Crappy which provider protocol to use.

For built-in providers, `id` and `api` are usually the same. For compatible gateways, proxies, or local servers, they can be different.

## Authentication

Providers usually read API keys from environment variables.

```sh
export OPENAI_API_KEY=...
```

You can store a key directly in settings with `auth.api_key`, but environment variables are safer for real credentials.

OAuth-backed subscriptions use a separate provider entry with an explicit driver:

```yaml
providers:
  - id: openai-subscription
    api: openai
    base_url: https://chatgpt.com/backend-api/codex
    auth:
      type: oauth
      driver: openai-codex
      client_id: app_EMoamEEZ73f0CkXaXp7hrann
      authorization_url: https://auth.openai.com/oauth/authorize
      token_url: https://auth.openai.com/oauth/token
      redirect_url: http://localhost:1455/auth/callback
      scopes:
        - openid
        - profile
        - email
        - offline_access
```

Connect or disconnect the subscription from the settings screen. Crappy stores and refreshes the credential for the configured provider; OAuth failures do not fall back to an API key.

## Custom Providers

Use a custom provider entry for compatible gateways, local model runtimes, proxies, or alternate credentials.

Custom providers work with any supported `api`: `anthropic`, `openai`, or `google`. The example below uses `openai` because many local runtimes expose an OpenAI-compatible endpoint.

```yaml
providers:
  - id: openai-local
    api: openai
    base_url: http://localhost:11434/v1
    auth:
      type: api_key
      api_key: local
```

Then select it in config:

```yaml
provider: openai-local
model: gemma4
```

The provider ID controls which model list appears in the settings screen. A provider with ID `openai-local` uses models listed for `openai-local`, not the built-in `openai` catalog.

## Custom Models

Add custom model metadata in `~/.crappy-ai/settings.yaml` with `models`.

The key under `models` should match the provider ID you select in config.

```yaml
providers:
  - id: openai-local
    api: openai
    base_url: http://localhost:11434/v1
    auth:
      type: api_key
      api_key: local

models:
  openai-local:
    - id: gemma4
      context_window: 131072
      output_limit: 8192
      capabilities:
        text: true
        tools: true
        streaming: true
```

Custom models are merged with Crappy's model catalog. If a custom model has the same `id` as a catalog model for the same provider, the settings entry wins.

You only need to include the fields you care about. `id` is required. Limits and capabilities help Crappy display better model details and reason about available context.

Common fields are:

- `id`
- `context_window`
- `input_limit`
- `output_limit`
- `capabilities`
- `cost`
- `knowledge_cutoff`
- `release_date`

## Unknown Models

You can use a model ID even when it is not in the catalog or settings.

```yaml
provider: openai-local
model: llama3.1:8b
```

Crappy will still try to call the selected provider with that model ID. When a model is unknown, Crappy has less metadata about context size, pricing, and capabilities.

In the settings screen, type the model ID into the model picker. If there is no matching model, press Enter to use the typed ID.

## Generation Settings

Generation settings control how the selected model produces each response.

Use `thinking` to choose the reasoning level passed to the model.

```yaml
thinking: high
```

Supported values are:

- `disabled`
- `low`
- `medium`
- `high`

Not every model treats thinking levels the same way. If a provider or model does not support a level, behavior depends on that provider.

Use `temperature` to control sampling randomness.

```yaml
temperature: 0.2
```

Use `max_output_tokens` to limit generated output per model call.

```yaml
max_output_tokens: 4096
```

`temperature` and `max_output_tokens` are optional. When omitted, Crappy leaves the value unset and the provider/model default applies.

## Model Metadata

Crappy ships with model metadata for common providers and refreshes it automatically when possible.

The metadata powers the settings screen and includes model IDs, context limits, output limits, pricing, and capabilities when available.

Configured models in `settings.yaml` are applied on top of this metadata, so local or custom models stay available even after the catalog refreshes.
