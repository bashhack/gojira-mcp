package main

import "github.com/mark3labs/mcp-go/mcp"

// Tool definitions for the gojira-mcp server. Each variable defines
// an MCP tool with its name, description, parameter schema, and
// annotation hints (read-only vs. destructive).

// createIssueTool is the compound "create and set up" tool — the primary
// value proposition of gojira-mcp. It combines issue creation with
// assignment, epic linkage, status transition, and sprint placement.
var createIssueTool = mcp.NewTool("create_issue",
	mcp.WithDescription("Create a Jira issue with optional assignment, epic linkage, status transition, and sprint placement — all in one call."),
	mcp.WithString("summary", mcp.Required(), mcp.Description("Issue title")),
	mcp.WithString("type", mcp.Required(), mcp.Description("Issue type: Task, Story, Bug, Epic, Subtask")),
	mcp.WithString("description", mcp.Description("Issue body (multiline supported)")),
	mcp.WithString("priority", mcp.Description("Priority: Highest, High, Medium, Low, Lowest")),
	mcp.WithString("assignee", mcp.Description("Assignee email or username")),
	mcp.WithString("epic", mcp.Description("Parent epic key, e.g. PROJ-100")),
	mcp.WithString("status", mcp.Description("Target status to transition to, e.g. 'In Progress'")),
	mcp.WithString("sprint", mcp.Description("Sprint ID, or 'active' to auto-detect the active sprint")),
	mcp.WithString("project", mcp.Description("Project key override")),
	mcp.WithArray("labels", mcp.Description("Labels to apply")),
)

var editIssueTool = mcp.NewTool("edit_issue",
	mcp.WithDescription("Update fields on an existing Jira issue."),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithString("summary", mcp.Description("New summary")),
	mcp.WithString("priority", mcp.Description("New priority")),
	mcp.WithString("assignee", mcp.Description("New assignee")),
	mcp.WithString("epic", mcp.Description("Parent epic key")),
	mcp.WithArray("labels", mcp.Description("Labels to add")),
)

var moveIssueTool = mcp.NewTool("move_issue",
	mcp.WithDescription("Transition a Jira issue to a new status."),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithString("status", mcp.Required(), mcp.Description("Target status, e.g. 'In Progress', 'Done'")),
)

var viewIssueTool = mcp.NewTool("view_issue",
	mcp.WithDescription("View details of a Jira issue."),
	mcp.WithReadOnlyHintAnnotation(true),
	mcp.WithDestructiveHintAnnotation(false),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithNumber("comments", mcp.Description("Number of comments to show (default 5)")),
)

var listIssuesTool = mcp.NewTool("list_issues",
	mcp.WithDescription("List or search Jira issues. Use jql for complex queries, or use the filter params for simple filtering."),
	mcp.WithReadOnlyHintAnnotation(true),
	mcp.WithDestructiveHintAnnotation(false),
	mcp.WithString("jql", mcp.Description("Raw JQL query (overrides other filters)")),
	mcp.WithString("assignee", mcp.Description("Filter by assignee")),
	mcp.WithString("type", mcp.Description("Filter by issue type")),
	mcp.WithString("priority", mcp.Description("Filter by priority")),
	mcp.WithString("parent", mcp.Description("Filter by parent/epic key")),
	mcp.WithString("project", mcp.Description("Project key override")),
	mcp.WithArray("status", mcp.Description("Filter by status (one or more)")),
	mcp.WithArray("labels", mcp.Description("Filter by labels")),
	mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
)

var addCommentTool = mcp.NewTool("add_comment",
	mcp.WithDescription("Add a comment to a Jira issue."),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithString("body", mcp.Required(), mcp.Description("Comment text")),
)

var addToSprintTool = mcp.NewTool("add_to_sprint",
	mcp.WithDescription("Add a Jira issue to a sprint."),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithString("sprint", mcp.Required(), mcp.Description("Sprint ID, or 'active' to auto-detect")),
)
