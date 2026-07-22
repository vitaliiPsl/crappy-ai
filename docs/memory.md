# Memory and Context

Crappy has two forms of persistent context: session history and explicit semantic memory.

## Persistent Memory

Crappy can remember durable information about you across sessions. Memories have one of three kinds:

- `profile` for stable facts about you and your world
- `preference` for choices that should influence responses
- `instruction` for persistent behavior you explicitly request

You can ask Crappy to remember something explicitly, for example:

```text
Remember that I prefer concise answers.
```

Crappy may also save a memory when you directly reveal something durable that is likely to improve future interactions. Instructions are never inferred and require an explicit request from you.

Crappy can list, correct, and forget saved memories. Memories are stored as structured JSON in `~/.crappy-ai/memory.json` by default. Change the location with `memory_path` in settings or `CRAPPY_MEMORY_PATH`.

Persistent memories may become outdated. Current requests, user-provided project instructions, and directly observed evidence take precedence. Crappy never edits `AGENTS.md` or `CLAUDE.md` to store memories.

Crappy does not derive memories from external content, guesses, or transient task details. Session transcripts remain the record of what happened in a conversation.

Crappy remembers work through session history.

When you continue a session, Crappy uses the conversation from that session as context for the next model turn. This includes the messages that matter for continuing the task, such as what you asked, what the assistant answered, and the results of tool work.

## Session Memory

Memory is scoped to the current session.

A new session starts with a clean context. An existing session continues from its saved history.

This means Crappy can keep track of:

- the task you are working on
- files and tools used during the session
- decisions and corrections from earlier turns
- relevant outputs from previous tool calls

Sessions are separate from each other. Starting a new session is the simplest way to start with a clean memory.

## Context

Before each turn, Crappy loads the session's message history and sends it to the model with the current system prompt, tools, and model settings.

Only conversation messages are used as model context. Other session activity, such as streaming progress, errors, and permission prompts, is kept in the session history but is not the main memory sent back to the model.

## Instruction Files

Crappy also reads `AGENTS.md` and `CLAUDE.md` files from the working directory and its ancestors. These files are included with the model instructions on each turn.

Use them for persistent guidance such as coding standards, build and test commands, project layout notes, and workflow preferences.

Instruction files are not session memory. They come from files on disk, are re-read for each run, and apply to any session started from the same directory tree.

## Compaction

Long sessions can become expensive or exceed the model's context window.

Compaction summarizes the current session and stores that summary in the conversation. After compaction, future turns continue from the latest summary plus any messages after it.

This keeps the important parts of the session while reducing how much old conversation needs to be sent back to the model.

## What Memory Is Not

Memory is not global across all sessions.

Crappy does not automatically carry details from one session into another. If something should matter in a new session, mention it again or keep working in the original session.

Memory is also not a permissions system. Permissions come from config and still decide which tools can run automatically, ask first, or be blocked. See [Permissions](permissions.md) for details.

## Related

See [Sessions](sessions.md) for how conversations are saved.

See [Configuration](configuration.md) for model, thinking, system prompt, and storage settings.
