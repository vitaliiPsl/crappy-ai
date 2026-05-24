# Tools

Tools are how Crappy reaches outside the chat.

They let Crappy inspect local files, make file changes, fetch web pages, and run shell commands when a task needs real context or action.

## Quick Examples

Ask Crappy to do work that needs tools:

```text
Read the README and summarize the setup steps.
```

```text
Update the config loader to support this new option.
```

```text
Run the Go tests for the permission package.
```

## Defaults

By default, Crappy can list and read files in the directory where it was started.

```yaml
permissions:
  default: ask
  allow:
    - tool: list
      pattern: "./**"
    - tool: read_file
      pattern: "./**"
```

File edits, shell commands, and web fetches ask first unless you add permission rules for them.

### `list`

List the contents of a directory.

Crappy uses this to explore directories before reading or editing files. It returns file and directory names, with directories shown using a trailing path separator.

The tool returns up to 100 entries by default and 200 at most.

### `read_file`

Read file contents with line numbers.

Crappy uses this to inspect docs, config, source files, notes, and other local text. It can read a whole file or a specific line range for larger files.

### `write_file`

Create a new file or overwrite an existing file.

Crappy uses this for new files or full-file replacements. For smaller changes to existing files, `edit_file` is usually safer.

If a file already exists, Crappy should read it before overwriting so it does not lose content accidentally.

### `edit_file`

Edit a file by replacing exact text.

Crappy uses this for precise changes to existing files. The text being replaced must match exactly.

By default, the text must match exactly one location in the file. For repeated changes, Crappy can replace every match.

To insert text, Crappy replaces nearby existing text with that same text plus the inserted content. To delete text, it replaces the matched text with an empty string.

### `web_fetch`

Fetch a web page and return a readable text snapshot.

Crappy uses this to inspect public documentation, articles, API pages, and other web references.

The URL must use HTTP or HTTPS. HTML pages are converted to readable text. Redirects are not followed automatically; redirect responses include the status and `Location` header.

By default, `web_fetch` returns up to 12,000 characters.

### `bash`

Run a shell command and return combined stdout and stderr.

Crappy runs commands through your shell. If `SHELL` is not set, it uses `sh`.

Use `bash` for tests, formatters, linters, project scripts, Git commands, and build tools.

Avoid long-running commands like servers and file watchers unless you are prepared to cancel them.

## Notes

Tools run on your machine with your normal system access.

File paths are interpreted from the directory where Crappy was started unless the assistant uses absolute paths.

Tool output is shown to the assistant and may be used as context for later steps.
