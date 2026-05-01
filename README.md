<p align="center">
  <img src="logo.png" alt="gojira-mcp" width="200">
</p>

<h1 align="center">gojira-mcp</h1>

<div align="center">

[![Tests](https://github.com/bashhack/gojira-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/bashhack/gojira-mcp/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/bashhack/gojira-mcp/graph/badge.svg)](https://codecov.io/gh/bashhack/gojira-mcp)
[![Go Reference](https://pkg.go.dev/badge/github.com/bashhack/gojira-mcp)](https://pkg.go.dev/github.com/bashhack/gojira-mcp)
[![Go Report Card](https://goreportcard.com/badge/github.com/bashhack/gojira-mcp)](https://goreportcard.com/report/github.com/bashhack/gojira-mcp)

</div>

An [MCP](https://modelcontextprotocol.io/) server that gives AI assistants structured access to Jira via [jira-cli](https://github.com/ankitpokhrel/jira-cli).

The headline feature is `create_issue` — a single tool call that creates an issue **and** assigns it, links it to an epic, transitions its status, and adds it to a sprint. No more chaining five shell commands with fragile escaping.

## vs. the official Atlassian MCP

Atlassian's [Rovo MCP Server](https://github.com/atlassian/atlassian-mcp-server) covers Jira, Confluence, and Compass — if you need breadth across Atlassian products, start there.

gojira-mcp wraps [jira-cli](https://github.com/ankitpokhrel/jira-cli), a mature CLI that already handles the full Jira workflow: sprints, epics, boards, cloning, linking, worklogs, custom fields, and both Cloud and Server/Data Center. gojira-mcp exposes all of that as structured MCP tools and adds compound operations — a single `create_issue` call that creates, assigns, links to an epic, transitions status, and adds to a sprint. No chaining, no escaping, no five separate API calls.

## Prerequisites

- [jira-cli](https://github.com/ankitpokhrel/jira-cli) installed and configured (`jira init`)

## Install

Download a prebuilt binary from [Releases](https://github.com/bashhack/gojira-mcp/releases):

Download the tarball and `checksums.txt` for your platform, then:

```bash
# Verify checksum (recommended)
shasum -a 256 -c checksums.txt

# macOS (Apple Silicon)
tar xzf gojira-mcp_darwin_arm64.tar.gz
sudo mv gojira-mcp /usr/local/bin/

# macOS (Intel)
tar xzf gojira-mcp_darwin_amd64.tar.gz
sudo mv gojira-mcp /usr/local/bin/

# Linux (x86_64)
tar xzf gojira-mcp_linux_amd64.tar.gz
sudo mv gojira-mcp /usr/local/bin/

# Linux (ARM64)
tar xzf gojira-mcp_linux_arm64.tar.gz
sudo mv gojira-mcp /usr/local/bin/
```

Or with Go 1.25+:

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

If you downloaded a prebuilt binary:

```bash
claude mcp add --scope user jira -- /usr/local/bin/gojira-mcp
```

If you installed with `go install`:

```bash
claude mcp add --scope user jira -- "$(go env GOPATH)/bin/gojira-mcp"
```

If you built from source:

```bash
claude mcp add --scope user jira -- ./gojira-mcp
```

Restart your Claude Code session after registering.

Verify it's connected:

```
/mcp
```

The `jira` server should show as healthy with 22 tools.

## Usage

Once registered, you don't call the tools directly — Claude Code calls them for you. Just ask for what you need in plain language:

```text
> Create a bug ticket "Login redirect broken on Safari", high priority, assign to alice@example.com

Created PROJ-456: "Login redirect broken on Safari"
  Status: To Do -- OK
  Sprint: 42 (active) -- OK
https://example.atlassian.net/browse/PROJ-456
```

```text
> What tickets are in progress for the PROJ project?

TYPE   KEY        SUMMARY                        STATUS        ASSIGNEE
Bug    PROJ-456   Login redirect broken on Safari In Progress   alice@example.com
Story  PROJ-411   Add dark mode support          In Progress   bob@example.com
```

```text
> Move PROJ-456 to Done and add a comment saying the fix shipped in v2.3.1

✓ Issue transitioned to state "Done"
✓ Comment added to issue "PROJ-456"
```

The tools work across any Jira project — whatever `jira-cli` is configured to access, gojira-mcp can reach.

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
| `components`  | string[] | no       | Components to set                               |
| `fix_version` | string[] | no       | Fix version(s) to set                           |
| `affects_version` | string[] | no   | Affects version(s) to set                       |
| `original_estimate` | string | no   | Time estimate, e.g. `2d 1h 30m`                |
| `custom`      | object   | no       | Custom fields as key-value pairs (see below)    |

Post-creation steps (status, sprint) are best-effort — failures are reported but don't prevent the issue from being created.

The `custom` parameter passes arbitrary fields to jira-cli's `--custom` flag. Field names must match your jira-cli configuration (see `jira init`). Common example — setting story points:

```text
> Create a story "Add caching layer" with 5 story points, assign to alice@example.com

# The tool receives: {"custom": {"story_points": "5"}}
```

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
| `body`     | string   | no       | New description   |
| `priority` | string   | no       | New priority      |
| `assignee` | string   | no       | New assignee      |
| `epic`     | string   | no       | Parent epic key   |
| `labels`   | string[] | no       | Labels to add     |
| `components` | string[] | no     | Components to replace |
| `fix_version` | string[] | no    | Fix version(s) to add |
| `affects_version` | string[] | no | Affects version(s) to add |
| `custom`   | object   | no       | Custom fields as key-value pairs |

### `clone_issue`

Clone a Jira issue, optionally overriding fields on the copy.

| Parameter    | Type     | Required | Description                    |
|--------------|----------|----------|--------------------------------|
| `key`        | string   | yes      | Issue key to clone             |
| `summary`    | string   | no       | Override summary               |
| `priority`   | string   | no       | Override priority              |
| `assignee`   | string   | no       | Override assignee              |
| `labels`     | string[] | no       | Override labels                |
| `components` | string[] | no       | Override components            |

### `delete_issue`

Delete a Jira issue. Use cascade to also delete subtasks.

| Parameter | Type   | Required | Description                          |
|-----------|--------|----------|--------------------------------------|
| `key`     | string | yes      | e.g. `PROJ-123`                      |
| `cascade` | bool   | no       | Also delete subtasks (default false) |

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

### `search_users`

Search for Jira users by name, email, or username. Useful for finding the right assignee without knowing their exact display name or email.

| Parameter | Type   | Required | Description                                        |
|-----------|--------|----------|----------------------------------------------------|
| `query`   | string | yes      | Partial name, email, or username                   |
| `project` | string | no       | Project key to scope to assignable users           |

This tool calls the Jira REST API directly (not jira-cli) and requires the `JIRA_API_TOKEN` environment variable to be set.

Example:

```text
> Search for users named "aaron" who can be assigned to PRO tickets

Aaron Maturen <aaron@example.com> (accountId: 5f3c...)
```

### `unlink_issues`

Remove a link between two Jira issues.

| Parameter | Type   | Required | Description                              |
|-----------|--------|----------|------------------------------------------|
| `inward`  | string | yes      | Issue key for the inward side            |
| `outward` | string | yes      | Issue key for the outward side           |

### `watch_issue`

Add a watcher to a Jira issue.

| Parameter | Type   | Required | Description                    |
|-----------|--------|----------|--------------------------------|
| `key`     | string | yes      | e.g. `PROJ-123`                |
| `watcher` | string | yes      | Watcher email or display name  |

### `add_worklog`

Log time spent on a Jira issue.

| Parameter      | Type   | Required | Description                    |
|----------------|--------|----------|--------------------------------|
| `key`          | string | yes      | e.g. `PROJ-123`                |
| `time_spent`   | string | yes      | e.g. `2d 1h 30m`              |
| `comment`      | string | no       | Worklog comment                |
| `started`      | string | no       | When work started              |
| `new_estimate` | string | no       | New remaining estimate         |

### `create_epic`

Create a Jira epic.

| Parameter    | Type     | Required | Description                    |
|--------------|----------|----------|--------------------------------|
| `name`       | string   | yes      | Epic name                      |
| `summary`    | string   | yes      | Epic summary                   |
| `body`       | string   | no       | Epic description               |
| `priority`   | string   | no       | Priority                       |
| `assignee`   | string   | no       | Assignee email or username     |
| `project`    | string   | no       | Project key override           |
| `labels`     | string[] | no       | Labels to apply                |
| `components` | string[] | no       | Components to set              |
| `custom`     | object   | no       | Custom fields                  |

### `list_epics`

List epics, or list issues within an epic.

| Parameter  | Type     | Required | Description                              |
|------------|----------|----------|------------------------------------------|
| `key`      | string   | no       | Epic key to list issues for              |
| `assignee` | string   | no       | Filter by assignee                       |
| `priority` | string   | no       | Filter by priority                       |
| `jql`      | string   | no       | Raw JQL query                            |
| `project`  | string   | no       | Project key override                     |
| `status`   | string[] | no       | Filter by status                         |
| `labels`   | string[] | no       | Filter by labels                         |
| `limit`    | number   | no       | Max results (default 50)                 |

### `add_issues_to_epic`

Add issues to an epic (max 50).

| Parameter | Type     | Required | Description                    |
|-----------|----------|----------|--------------------------------|
| `epic`    | string   | yes      | Epic key, e.g. `PROJ-100`     |
| `issues`  | string[] | yes      | Issue keys to add              |

### `remove_issues_from_epic`

Remove the epic link from issues (max 50).

| Parameter | Type     | Required | Description                    |
|-----------|----------|----------|--------------------------------|
| `issues`  | string[] | yes      | Issue keys to remove           |

### `list_sprints`

List sprints in the project board.

| Parameter | Type   | Required | Description                              |
|-----------|--------|----------|------------------------------------------|
| `state`   | string | no       | Filter: `future`, `active`, `closed`     |
| `project` | string | no       | Project key override                     |

### `list_boards`

List Jira boards.

| Parameter | Type   | Required | Description                    |
|-----------|--------|----------|--------------------------------|
| `project` | string | no       | Project key override           |

### `list_projects`

List Jira projects. No parameters.

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
