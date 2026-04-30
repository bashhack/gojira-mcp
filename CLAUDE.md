# CLAUDE.md — gojira-mcp

## What This Is

An MCP server in Go that wraps jira-cli, giving AI assistants structured Jira tools. Single binary, no daemon, communicates over stdio via JSON-RPC 2.0.

## Project Layout

```
main.go          — entry point, server setup, tool registration
jira.go          — jira-cli exec wrapper, output parsing
tools.go         — MCP tool definitions (schemas, descriptions, annotations)
handlers.go      — tool handler functions
```

Everything is `package main`. No internal packages.

## Development

```bash
make run/tests          # run test suite
make run/tests/race     # run with race detection
make lint               # format + vet + staticcheck
make lint/golangci      # golangci-lint v2.6.2
make lint/fieldalignment # struct field alignment check (non-destructive)
make format             # goimports
make format/check       # check formatting without modifying
make audit              # tidy + verify + format + vet + staticcheck + tests
make pre-commit         # what the git hook runs
make build              # build to ./bin/gojira-mcp
make build/optimize     # build stripped binary
make install            # go install to GOPATH/bin
```

## Testing

Tests use a `jiraRunner` function variable swapped with fakes to avoid shelling out to jira-cli. Two helpers:

- `fakeRunner(stdout, stderr, err)` — returns fixed responses
- `sequenceRunner` — records calls and returns responses in order, for multi-step handlers like `create_issue`

Tests must not use `t.Parallel()` while swapping `jiraRunner`.

Use `map[string]struct{}` for table-driven tests. Named fields only.

## Adding a New Tool

1. Define the tool schema in `tools.go` using `mcp.NewTool()`
2. Write the handler in `handlers.go` — validate required params, call `jiraRunner`, return `mcp.NewToolResultText()` or `mcp.NewToolResultError()`
3. Register in `main.go` with `s.AddTool()`
4. Add `//nolint:gocritic` on the handler signature (mcp-go requires `CallToolRequest` by value)
5. Write tests in `handlers_test.go`

## Conventions

- No section separator comments in code
- No AI attribution in commits or PRs
- Single-line commit messages, no body
- golangci-lint v2.6.2 must pass before merge
- Read-only tools get `WithReadOnlyHintAnnotation(true)` and `WithDestructiveHintAnnotation(false)`
- Module path: `github.com/bashhack/gojira-mcp`
- goimports `-local github.com/bashhack/gojira-mcp`

## Dependencies

- `github.com/mark3labs/mcp-go` — MCP protocol SDK
- `jira-cli` — must be installed and configured on the host (`jira init`)

## CI

GitHub Actions: test (Go 1.25), build, lint (golangci-lint v2.6.2), codecov upload. Branch protection requires all checks green + 1 approving review.
