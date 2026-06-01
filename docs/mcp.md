# MCP

MCP (Model Context Protocol) lets Crappy use tools from external servers.

An MCP server exposes tools that Crappy can call alongside its built-in ones, so the assistant can reach systems like issue trackers, databases, documentation, or your own services.

## Quick Example

Add a server to the settings file under `mcp`:

```yaml
mcp:
  - name: deepwiki
    type: http
    url: https://mcp.deepwiki.com/mcp
```

The next time Crappy starts, that server's tools become available to the assistant. Ask for work that needs them:

```text
Use deepwiki to summarize the architecture of the facebook/react repo.
```

## Where MCP Lives

MCP servers are configured in the settings file, because they describe how Crappy is wired up on your machine.

```text
~/.crappy-ai/settings.yaml
```

Each entry under `mcp` is one server. A server has a `name` and a transport.

```yaml
mcp:
  - name: docs
    type: http
    url: https://mcp.deepwiki.com/mcp
  - name: filesystem
    command: npx
    args:
      - -y
      - "@modelcontextprotocol/server-filesystem"
      - /path/to/dir
```

`name` identifies the server and prefixes its tools.

`type` selects the transport: `stdio` or `http`. It defaults to `stdio`.

## Transports

### stdio

A stdio server runs as a local process and talks to Crappy over its standard input and output. Use it for local tools and custom scripts.

```yaml
mcp:
  - name: filesystem
    type: stdio
    command: npx
    args:
      - -y
      - "@modelcontextprotocol/server-filesystem"
      - /path/to/dir
    env:
      - LOG_LEVEL=info
```

`command` is the executable to run.

`args` are passed to the command.

`env` adds environment variables to the process. Each entry is `KEY=value`. The process also inherits Crappy's own environment.

### http

An http server is a remote endpoint Crappy connects to over streamable HTTP. Use it for hosted services.

```yaml
mcp:
  - name: docs
    type: http
    url: https://mcp.deepwiki.com/mcp
```

`url` is the server's endpoint.

## Authentication

Remote servers often need a credential. Crappy sends it as an HTTP header.

Set a header directly with `headers`:

```yaml
mcp:
  - name: internal
    type: http
    url: https://mcp.internal.example.com
    auth:
      headers:
        Authorization: "Bearer static-token"
        X-Tenant: acme
```

Or read the header value from an environment variable with `header_env`, which keeps secrets out of the settings file:

```yaml
mcp:
  - name: github
    type: http
    url: https://api.githubcopilot.com/mcp/
    auth:
      header_env:
        Authorization: GITHUB_MCP_TOKEN
```

`headers` maps a header name to a literal value.

`header_env` maps a header name to the environment variable that holds its value. The variable's value is used as the full header, so set it accordingly:

```sh
export GITHUB_MCP_TOKEN="Bearer ghp_..."
```

If a referenced environment variable is empty, the server fails to connect and reports the missing variable.

## Tools

Crappy loads a server's tools when it connects and namespaces each one with the server name:

```text
mcp__<server>__<tool>
```

For example, the `search` tool on a server named `github` becomes `mcp__github__search`. The namespace keeps tools from different servers from colliding and makes it clear where a tool came from.

Tool calls follow the same [permission rules](permissions.md) as built-in tools.

If a server announces that its tools changed, Crappy refreshes them automatically without a restart.

## Connection and Status

Crappy connects to MCP servers in the background at startup, so it never blocks on a slow or unreachable server.

A server moves through these states:

- `disconnected` — not connected yet
- `connecting` — establishing the connection
- `connected` — ready, tools available
- `failed` — the last attempt errored

If a connection drops or a call fails, the server moves to `failed`. A failed server shows its error so you can see why it could not connect.

## See Also

- [Configuration](configuration.md) for the settings file and precedence.
- [Tools](tools.md) for how the assistant uses tools.
- [Permissions](permissions.md) for controlling which tool calls run.
