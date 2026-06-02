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

Remote servers often need credentials. Crappy supports static HTTP headers and OAuth for HTTP MCP servers.

### Headers

Use static headers when the server expects an API key, bearer token, tenant ID, or another fixed request header.

Set a header directly with `headers`:

```yaml
mcp:
  - name: internal
    type: http
    url: https://mcp.internal.example.com
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
    header_env:
      Authorization: GITHUB_MCP_TOKEN
```

`headers` maps a header name to a literal value.

`header_env` maps a header name to the environment variable that holds its value. The variable's value is used as the full header, so set it accordingly:

```sh
export GITHUB_MCP_TOKEN="Bearer ghp_..."
```

If a referenced environment variable is empty, the server fails to connect and reports the missing variable.

Static headers are added to MCP HTTP requests unless that header is already set by the MCP transport. This lets OAuth-provided headers, such as `Authorization`, take precedence over static fallback headers.

### OAuth

HTTP MCP servers that support OAuth can use an authorization-code flow. OAuth is passive during startup: Crappy detects that authorization is needed, but it does not open a browser until you ask it to.

Enable OAuth on the server:

```yaml
mcp:
  - name: sentry
    type: http
    url: https://mcp.sentry.dev/mcp
    oauth:
      enabled: true
```

When the server asks for authorization, Crappy marks the MCP client as `auth required`. Open the MCP clients screen and press `a` on that client to start the OAuth flow. Crappy then starts a local callback server, opens the authorization URL in your browser, and retries the MCP connection after the callback completes. The callback server only lives for that one authentication attempt and shuts down after success, failure, or cancellation.

By default the callback URL is:

```text
http://127.0.0.1:14545/oauth/callback
```

You can customize it:

```yaml
mcp:
  - name: sentry
    type: http
    url: https://mcp.sentry.dev/mcp
    oauth:
      callback_host: 127.0.0.1
      callback_port: 14546
```

When you press `a` to authenticate, Crappy opens the authorization URL in your browser. If the browser cannot be opened, the authentication attempt fails and the MCP client reports the error.

Crappy supports three OAuth client registration modes.

With dynamic registration, the authorization server gives Crappy a client ID during the OAuth flow. This is the default when no `client_id` or `client_id_metadata_url` is configured:

```yaml
mcp:
  - name: sentry
    type: http
    url: https://mcp.sentry.dev/mcp
    oauth:
      enabled: true
```

For a pre-registered OAuth client, provide the client ID and, if needed, a client secret:

```yaml
mcp:
  - name: internal
    type: http
    url: https://mcp.internal.example.com/mcp
    oauth:
      client_id: crappy-local
      client_secret_env: INTERNAL_MCP_CLIENT_SECRET
      redirect_url: http://127.0.0.1:14545/oauth/callback
```

For an OAuth server that supports Client ID Metadata Documents, provide the metadata document URL:

```yaml
mcp:
  - name: internal
    type: http
    url: https://mcp.internal.example.com/mcp
    oauth:
      client_id_metadata_url: https://example.com/crappy/oauth-client.json
      redirect_url: http://127.0.0.1:14545/oauth/callback
```

Set `dynamic_registration: false` if the server must not try dynamic registration.

## Timeouts

Crappy bounds how long it waits on a server with two separate budgets, because connecting and running a tool are different kinds of wait:

- `connect_timeout` bounds establishing the connection.
- `request_timeout` bounds each request on a live connection — listing tools or running a tool call.

```yaml
mcp:
  - name: docs
    type: http
    url: https://mcp.deepwiki.com/mcp
    connect_timeout: 10s
    request_timeout: 30s
```

Each value is a duration string such as `500ms`, `10s`, or `2m`, and applies per operation, so every request gets the full budget. When a timeout is unset, that operation is not time-bounded and waits as long as the server takes.

Keeping them separate lets a connection fail fast while a slow tool call still gets a generous budget. If an operation exceeds its timeout, that request fails and the server moves to `failed`.

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
- `auth_required` — the server needs OAuth; authenticate it from the MCP clients screen
- `failed` — the last attempt errored

If a connection drops or a non-auth call fails, the server moves to `failed`. If an OAuth-enabled server asks for authorization, the server moves to `auth_required` instead. A failed server shows its error so you can see why it could not connect.

## See Also

- [Configuration](configuration.md) for the settings file and precedence.
- [Tools](tools.md) for how the assistant uses tools.
- [Permissions](permissions.md) for controlling which tool calls run.
