# Sessions

Sessions keep a conversation and its context so Crappy can continue the same thread later.

A session stores basic metadata, such as its ID, title, working directory, usage, and timestamps. It also stores the conversation history as events.

## How Sessions Work

Crappy starts with a draft session.

The draft becomes a saved session when you send the first message. Crappy uses that first message to create a short title.

When you continue a saved session, Crappy loads the session history and uses it as model context. This lets the assistant remember what you were working on, what files or tools were used, and what decisions were made earlier.

Sessions are separate from each other. Start a new session for a different task, project, or thread of work.

## Events

Session history is recorded as events.

Events include messages, streamed assistant output, tool activity, permission prompts, errors, completed turns, and cancelled turns.

The most important event is `message`, because message events are what Crappy uses as conversation context when continuing a session.

Streaming events such as `content_started`, `content_delta`, and `content_done` preserve what happened while the assistant was responding. Other events, such as `turn_complete`, `turn_cancelled`, `error`, and `permission_prompt`, record the state around a turn.

## Context

Crappy uses the session's message history as context for future turns in that session.

If a session becomes long, Crappy can compact it by adding a summary. Future turns continue from the latest summary plus any messages after it.

Compaction keeps the important context while reducing how much previous conversation needs to be sent back to the model.

## Storage

Sessions are stored in the directory configured by `sessions_dir` in `~/.crappy-ai/settings.yaml`.

```yaml
sessions_dir: ~/.crappy-ai/sessions
```

You can also override it with an environment variable:

```sh
CRAPPY_SESSIONS_DIR=~/.local/share/crappy/sessions
```

See [Configuration](configuration.md) for more settings options.

## Notes

Sessions remember the working directory where they were created.

Permissions are not scoped to a single session. They come from config and apply globally. See [Permissions](permissions.md) for details.
