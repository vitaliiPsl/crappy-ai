# Permissions

Control which tool calls Crappy can run automatically, which ones need approval, and which ones are blocked.

Each tool call resolves to one of:

- `allow`: run the tool without asking.
- `ask`: prompt for approval.
- `deny`: block the action.

Permissions are enforced by Crappy. Instructions can guide what the assistant tries to do, but they do not override permission rules.

## Quick Config

Use the `permissions` config to set a default and then override specific tools or patterns. By default, Crappy stores config in `~/.crappy-ai/config.yaml`.

```yaml
mode: default
permissions:
  default: ask
  deny:
    - tool: bash
      pattern: "rm *"
  ask:
    - tool: web_fetch
      pattern: "domain:github.com"
  allow:
    - tool: list
      pattern: "./**"
    - tool: read_file
      pattern: "./**"
```

Rules have a `tool` and an optional `pattern`. A rule without a pattern, or with `pattern: "*"`, matches every use of that tool.

## Defaults

The default config sets `default: ask` and allows read-only exploration in the directory where Crappy was started:

```yaml
mode: default
permissions:
  default: ask
  allow:
    - tool: list
      pattern: "./**"
    - tool: read_file
      pattern: "./**"
```

That means Crappy can list and read files under the current working directory without prompting, while edits, shell commands, and web fetches ask first unless you add rules for them.

## Modes

Permission modes control how rules are applied.

```yaml
mode: default
```

`default` uses your configured permission rules.

```yaml
mode: yolo
```

`yolo` allows all tool calls without prompting. Use it only when you trust the assistant and the workspace.

You can use yolo mode for a single run:

```sh
crappy -mode yolo
```

Or through an environment variable:

```sh
CRAPPY_MODE=yolo
```

## Configuration

You can allow, ask, or deny a whole tool:

```yaml
mode: default

permissions:
  default: ask
  allow:
    - tool: list
    - tool: read_file
  deny:
    - tool: write_file
    - tool: edit_file
```

You can also target a specific input pattern:

```yaml
permissions:
  allow:
    - tool: bash
      pattern: "git status"
    - tool: web_fetch
      pattern: "domain:example.com"
  deny:
    - tool: bash
      pattern: "git push *"
```

Rules are evaluated in this order:

1. `deny`
2. `ask`
3. `allow`
4. `default`

The first matching decision wins. A deny rule always beats an ask or allow rule. An ask rule beats an allow rule.

## Available Permissions

| Tool | What the pattern matches |
| --- | --- |
| `list` | Directory path |
| `read_file` | File path |
| `write_file` | File path |
| `edit_file` | File path |
| `web_fetch` | URL or domain |
| `bash` | Shell command |
| `mcp__<server>__<tool>` | Matched by tool name; see [MCP Permissions](#mcp-permissions) |

## Path Permissions

Path rules apply to `list`, `read_file`, `write_file`, and `edit_file`.

```yaml
permissions:
  allow:
    - tool: read_file
      pattern: "./docs/**"
    - tool: edit_file
      pattern: "./src/**/*.go"
  deny:
    - tool: read_file
      pattern: "/etc/**"
```

Path patterns are converted to absolute paths before matching. Relative patterns are resolved from the directory where Crappy was started.

Supported wildcards:

- `*` matches within one path segment.
- `?` matches one character.
- `[abc]` and `[a-z]` match character classes.
- `**` as a whole segment matches zero or more path segments.

Examples:

```yaml
permissions:
  allow:
    - tool: read_file
      pattern: "./README.md"
    - tool: read_file
      pattern: "./docs/**"
    - tool: edit_file
      pattern: "./src/**/*.go"
```

## URL Permissions

URL rules apply to `web_fetch`.

Crappy supports two URL pattern forms:

```yaml
permissions:
  allow:
    - tool: web_fetch
      pattern: "url:https://example.com/docs"
    - tool: web_fetch
      pattern: "domain:example.com"
    - tool: web_fetch
      pattern: "domain:*.example.com"
```

`url:` rules match the raw URL exactly, after trimming surrounding whitespace.

`domain:` rules match the URL hostname. They ignore ports, lowercase the domain, and support wildcards:

- `domain:example.com` matches `https://example.com/page`.
- `domain:example.com` does not match subdomains.
- `domain:*.example.com` matches `https://api.example.com`.
- `domain:**.example.com` matches deeper subdomains like `https://v1.api.example.com`.
- `domain:*` matches any non-empty hostname.

Bare URL globs such as `https://example.com/*` are not supported. Use `domain:` rules for host-wide access.

## Bash Permissions

Bash rules apply to the `bash` tool's `command` argument.

```yaml
permissions:
  allow:
    - tool: bash
      pattern: "go test *"
    - tool: bash
      pattern: "go vet *"
  deny:
    - tool: bash
      pattern: "rm *"
```

Exact patterns match the full trimmed command:

```yaml
permissions:
  allow:
    - tool: bash
      pattern: "npm run build"
```

Wildcard patterns support:

- `*` for zero or more characters.
- `?` for one character.

The pattern is matched against the whole command, not just the first word.

### Compound Commands

Compound commands are checked command by command. For example:

```sh
go test ./... && go vet ./...
```

is treated as two commands:

```text
go test ./...
go vet ./...
```

A compound command is allowed by command-pattern rules only when every command in it is allowed. You can also allow the exact full command, or allow the whole `bash` tool.

This prevents a broad rule like:

```yaml
- tool: bash
  pattern: "echo *"
```

from automatically allowing:

```sh
echo ok && rm -rf tmp
```

Deny and ask rules are also checked against each command in the chain, so `rm *` can block the second command in a compound command.

### Substitutions

Commands with shell substitutions are treated cautiously:

```sh
git $(rm -rf /)
diff <(ls a) <(ls b)
```

Broad command-pattern rules like `git *` do not automatically approve commands with substitutions. Deny and ask rules can still match commands inside substitutions.

### Background Bash

Background `bash` calls use the same permission rules as foreground `bash` calls. The `background` argument only changes execution scheduling; it does not change permission matching.

See [Background Execution](background.md) for job tools and lifecycle details.

## MCP Permissions

Tools from [MCP servers](mcp.md) are named `mcp__<server>__<tool>`, such as `mcp__github__search`. They are matched by the `tool` field itself, which supports wildcards, so the `pattern` field is not used.

```yaml
permissions:
  allow:
    - tool: mcp__github__search   # one tool
    - tool: mcp__github__*        # every tool from the github server
  deny:
    - tool: mcp__*                # every MCP tool
```

- `mcp__github__search` matches one tool.
- `mcp__github__*` matches every tool from the `github` server.
- `mcp__*` matches every MCP tool.

When an MCP tool prompts for approval, Crappy offers to allow just that tool or every tool from its server, so you can grant a whole server at once.

## Approval Prompts

When a tool call resolves to `ask`, Crappy prompts you for a decision.

| Key | Action |
| --- | --- |
| `y` or Enter | Allow once |
| `e` | Allow exact, when available |
| `g` | Allow pattern, when available |
| `n` or Esc | Deny |

`Allow once` and `Deny` apply only to the current tool call.

`Allow exact` and `Allow pattern` save a global allow rule. Future matching tool calls run without prompting unless a deny or ask rule takes precedence.

## Notes

Permissions are not a sandbox. If you allow a shell command, it runs with your normal system permissions.

Permissions apply globally from your config. They are not scoped to a single chat session.
