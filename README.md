<p align="center">
  <img src="logo.png" alt="gojira-mcp" width="200">
</p>

<h1 align="center">gojira-mcp</h1>

An [MCP](https://modelcontextprotocol.io/) server that gives AI assistants structured access to Jira via [jira-cli](https://github.com/ankitpokhrel/jira-cli).

The headline feature is `create_issue` — a single tool call that creates an issue **and** assigns it, links it to an epic, transitions its status, and adds it to a sprint. No more chaining five shell commands with fragile escaping.

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [jira-cli](https://github.com/ankitpokhrel/jira-cli) installed and configured (`jira init`)

## Install

```bash
go install github.com/bashhack/gojira-mcp@latest
```

Or build from source:

```bash
git clone https://github.com/bashhack/gojira-mcp.git
cd gojira-mcp
go build -o gojira-mcp .
```

## Register with Claude Code

```bash
claude mcp add --scope user jira -- /path/to/gojira-mcp
```

Verify it's connected:

```
/mcp
```

The `jira` server should show as healthy with 7 tools.

## Tools

### `create_issue`

Create a Jira issue with optional assignment, epic linkage, status transition, and sprint placement — all in one call.

| Parameter     | Type     | Required | Description                                     |
|---------------|----------|----------|-------------------------------------------------|
| `summary`     | string   | yes      | Issue title                                     |
| `type`        | string   | yes      | Task, Story, Bug, Epic, Subtask                 |
| `description` | string   | no       | Issue body (multiline supported)                |
| `priority`    | string   | no       | Highest, High, Medium, Low, Lowest              |
| `assignee`    | string   | no       | Email or username                               |
| `epic`        | string   | no       | Parent epic key, e.g. `PROJ-100`                |
| `status`      | string   | no       | Target status, e.g. `In Progress`               |
| `sprint`      | string   | no       | Sprint ID, or `active` to auto-detect           |
| `project`     | string   | no       | Project key override                            |
| `labels`      | string[] | no       | Labels to apply                                 |

Post-creation steps (status, sprint) are best-effort — failures are reported but don't prevent the issue from being created.

Example output:

```
Created PROJ-456: "Implement caching layer"
  Status: In Progress -- OK
  Sprint: 3601 (active) -- OK
https://example.atlassian.net/browse/PROJ-456
```

### `edit_issue`

Update fields on an existing issue.

| Parameter  | Type     | Required | Description       |
|------------|----------|----------|-------------------|
| `key`      | string   | yes      | e.g. `PROJ-123`   |
| `summary`  | string   | no       | New summary       |
| `priority` | string   | no       | New priority      |
| `assignee` | string   | no       | New assignee      |
| `epic`     | string   | no       | Parent epic key   |
| `labels`   | string[] | no       | Labels to add     |

### `move_issue`

Transition an issue to a new status.

| Parameter | Type   | Required | Description                    |
|-----------|--------|----------|--------------------------------|
| `key`     | string | yes      | e.g. `PROJ-123`                |
| `status`  | string | yes      | e.g. `In Progress`, `Done`     |

### `view_issue`

View issue details (read-only).

| Parameter  | Type   | Required | Description                      |
|------------|--------|----------|----------------------------------|
| `key`      | string | yes      | e.g. `PROJ-123`                  |
| `comments` | number | no       | Number of comments (default 5)   |

### `list_issues`

Search issues with filters or raw JQL (read-only).

| Parameter  | Type     | Required | Description                              |
|------------|----------|----------|------------------------------------------|
| `jql`      | string   | no       | Raw JQL (overrides other filters)        |
| `assignee` | string   | no       | Filter by assignee                       |
| `type`     | string   | no       | Filter by issue type                     |
| `priority` | string   | no       | Filter by priority                       |
| `parent`   | string   | no       | Filter by parent/epic key                |
| `project`  | string   | no       | Project key override                     |
| `status`   | string[] | no       | Filter by status                         |
| `labels`   | string[] | no       | Filter by labels                         |
| `limit`    | number   | no       | Max results (default 20)                 |

### `add_comment`

Add a comment to an issue.

| Parameter | Type   | Required | Description     |
|-----------|--------|----------|-----------------|
| `key`     | string | yes      | e.g. `PROJ-123` |
| `body`    | string | yes      | Comment text    |

### `add_to_sprint`

Add an issue to a sprint.

| Parameter | Type   | Required | Description                         |
|-----------|--------|----------|-------------------------------------|
| `key`     | string | yes      | e.g. `PROJ-123`                     |
| `sprint`  | string | yes      | Sprint ID, or `active` to auto-detect |

## How it works

gojira-mcp communicates over stdio using the [MCP protocol](https://modelcontextprotocol.io/) (JSON-RPC 2.0). It wraps [jira-cli](https://github.com/ankitpokhrel/jira-cli) commands via `os/exec` — each operation is a separate process, no shell involved, so multiline descriptions and special characters work without escaping issues.

Diagnostic logs go to stderr (as required by MCP). stdout is reserved for JSON-RPC messages.

## Development

```bash
# Run tests
go test -v -race ./...

# Build
go build -o gojira-mcp .

# Smoke test the MCP protocol
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}' | ./gojira-mcp 2>/dev/null
```

## License

MIT
