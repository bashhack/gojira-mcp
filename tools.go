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
	mcp.WithArray("components", mcp.Description("Components to set")),
	mcp.WithArray("fix_version", mcp.Description("Fix version(s) to set")),
	mcp.WithArray("affects_version", mcp.Description("Affects version(s) to set")),
	mcp.WithString("original_estimate", mcp.Description("Time estimate, e.g. '2d 1h 30m'")),
	mcp.WithObject("custom", mcp.Description("Custom fields as key-value pairs, e.g. {\"story-points\": \"5\"}"), mcp.AdditionalProperties(true)),
)

var editIssueTool = mcp.NewTool("edit_issue",
	mcp.WithDescription("Update fields on an existing Jira issue."),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithString("summary", mcp.Description("New summary")),
	mcp.WithString("body", mcp.Description("New description")),
	mcp.WithString("priority", mcp.Description("New priority")),
	mcp.WithString("assignee", mcp.Description("New assignee")),
	mcp.WithString("epic", mcp.Description("Parent epic key")),
	mcp.WithArray("labels", mcp.Description("Labels to add")),
	mcp.WithArray("components", mcp.Description("Components to replace")),
	mcp.WithArray("fix_version", mcp.Description("Fix version(s) to add")),
	mcp.WithArray("affects_version", mcp.Description("Affects version(s) to add")),
	mcp.WithObject("custom", mcp.Description("Custom fields as key-value pairs, e.g. {\"story-points\": \"5\"}"), mcp.AdditionalProperties(true)),
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
	mcp.WithString("project", mcp.Description("Project key — required when using 'active' with a non-default project")),
)

var assignIssueTool = mcp.NewTool("assign_issue",
	mcp.WithDescription("Assign a Jira issue to a user, or unassign it."),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithString("assignee", mcp.Required(), mcp.Description("Assignee email or username, or 'none' to unassign")),
)

var linkIssuesTool = mcp.NewTool("link_issues",
	mcp.WithDescription("Create a link between two Jira issues."),
	mcp.WithString("inward", mcp.Required(), mcp.Description("Issue key for the inward side, e.g. PROJ-123")),
	mcp.WithString("outward", mcp.Required(), mcp.Description("Issue key for the outward side, e.g. PROJ-456")),
	mcp.WithString("type", mcp.Required(), mcp.Description("Link type, e.g. 'Blocks', 'is blocked by', 'Duplicate', 'Relates'")),
)

var searchUsersTool = mcp.NewTool("search_users",
	mcp.WithDescription("Search for Jira users by name, email, or username. Returns display names and emails for use in assignment."),
	mcp.WithReadOnlyHintAnnotation(true),
	mcp.WithDestructiveHintAnnotation(false),
	mcp.WithString("query", mcp.Required(), mcp.Description("Search string: partial name, email, or username")),
	mcp.WithString("project", mcp.Description("Project key to scope results to assignable users")),
)

var cloneIssueTool = mcp.NewTool("clone_issue",
	mcp.WithDescription("Clone a Jira issue, optionally overriding fields on the copy."),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key to clone, e.g. PROJ-123")),
	mcp.WithString("summary", mcp.Description("Override summary on the clone")),
	mcp.WithString("priority", mcp.Description("Override priority")),
	mcp.WithString("assignee", mcp.Description("Override assignee")),
	mcp.WithArray("labels", mcp.Description("Override labels")),
	mcp.WithArray("components", mcp.Description("Override components")),
)

var deleteIssueTool = mcp.NewTool("delete_issue",
	mcp.WithDescription("Delete a Jira issue. Use cascade to also delete subtasks."),
	mcp.WithDestructiveHintAnnotation(true),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithBoolean("cascade", mcp.Description("Also delete subtasks (default false)")),
)

var unlinkIssuesTool = mcp.NewTool("unlink_issues",
	mcp.WithDescription("Remove a link between two Jira issues."),
	mcp.WithString("inward", mcp.Required(), mcp.Description("Issue key for the inward side, e.g. PROJ-123")),
	mcp.WithString("outward", mcp.Required(), mcp.Description("Issue key for the outward side, e.g. PROJ-456")),
)

var watchIssueTool = mcp.NewTool("watch_issue",
	mcp.WithDescription("Add a watcher to a Jira issue."),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithString("watcher", mcp.Required(), mcp.Description("Watcher email or display name")),
)

var addWorklogTool = mcp.NewTool("add_worklog",
	mcp.WithDescription("Log time spent on a Jira issue."),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithString("time_spent", mcp.Required(), mcp.Description("Time spent, e.g. '2d 1h 30m'")),
	mcp.WithString("comment", mcp.Description("Worklog comment")),
	mcp.WithString("started", mcp.Description("When work started (datetime string)")),
	mcp.WithString("new_estimate", mcp.Description("New remaining estimate")),
)

var createEpicTool = mcp.NewTool("create_epic",
	mcp.WithDescription("Create a Jira epic."),
	mcp.WithString("name", mcp.Required(), mcp.Description("Epic name")),
	mcp.WithString("summary", mcp.Required(), mcp.Description("Epic summary")),
	mcp.WithString("body", mcp.Description("Epic description")),
	mcp.WithString("priority", mcp.Description("Priority")),
	mcp.WithString("assignee", mcp.Description("Assignee email or username")),
	mcp.WithString("project", mcp.Description("Project key override")),
	mcp.WithArray("labels", mcp.Description("Labels to apply")),
	mcp.WithArray("components", mcp.Description("Components to set")),
	mcp.WithObject("custom", mcp.Description("Custom fields as key-value pairs"), mcp.AdditionalProperties(true)),
)

var listEpicsTool = mcp.NewTool("list_epics",
	mcp.WithDescription("List epics, or list issues in an epic."),
	mcp.WithReadOnlyHintAnnotation(true),
	mcp.WithDestructiveHintAnnotation(false),
	mcp.WithString("key", mcp.Description("Epic key to list issues for (omit to list epics)")),
	mcp.WithString("assignee", mcp.Description("Filter by assignee")),
	mcp.WithString("priority", mcp.Description("Filter by priority")),
	mcp.WithString("jql", mcp.Description("Raw JQL query")),
	mcp.WithString("project", mcp.Description("Project key override")),
	mcp.WithArray("status", mcp.Description("Filter by status")),
	mcp.WithArray("labels", mcp.Description("Filter by labels")),
	mcp.WithNumber("limit", mcp.Description("Max results (default 50)")),
)

var addIssuesToEpicTool = mcp.NewTool("add_issues_to_epic",
	mcp.WithDescription("Add issues to an epic (max 50)."),
	mcp.WithString("epic", mcp.Required(), mcp.Description("Epic key, e.g. PROJ-100")),
	mcp.WithArray("issues", mcp.Required(), mcp.Description("Issue keys to add")),
)

var removeIssuesFromEpicTool = mcp.NewTool("remove_issues_from_epic",
	mcp.WithDescription("Remove the epic link from issues (max 50)."),
	mcp.WithArray("issues", mcp.Required(), mcp.Description("Issue keys to remove from their epic")),
)

var listSprintsTool = mcp.NewTool("list_sprints",
	mcp.WithDescription("List sprints in the project board."),
	mcp.WithReadOnlyHintAnnotation(true),
	mcp.WithDestructiveHintAnnotation(false),
	mcp.WithString("state", mcp.Description("Filter by state: future, active, closed (comma-separated)")),
	mcp.WithString("project", mcp.Description("Project key override")),
)

var listBoardsTool = mcp.NewTool("list_boards",
	mcp.WithDescription("List Jira boards."),
	mcp.WithReadOnlyHintAnnotation(true),
	mcp.WithDestructiveHintAnnotation(false),
	mcp.WithString("project", mcp.Description("Project key override")),
)

var listProjectsTool = mcp.NewTool("list_projects",
	mcp.WithDescription("List Jira projects."),
	mcp.WithReadOnlyHintAnnotation(true),
	mcp.WithDestructiveHintAnnotation(false),
)

var changeIssueTypeTool = mcp.NewTool("change_issue_type",
	mcp.WithDescription("Change the issue type of an existing Jira issue (e.g. Task → Story, Bug → Task). Calls the Jira REST API directly because jira-cli does not expose this operation."),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithString("type", mcp.Required(), mcp.Description("Target issue type name, e.g. 'Story', 'Task', 'Bug', 'Epic'")),
)

var moveIssueToProjectTool = mcp.NewTool("move_issue_to_project",
	mcp.WithDescription("Move a Jira issue to a different project. Note: Jira Cloud rejects cross-project moves when the source workflow or issue type is not configured in the target project — fall back to the Jira UI's Move action if that happens."),
	mcp.WithString("key", mcp.Required(), mcp.Description("Issue key, e.g. PROJ-123")),
	mcp.WithString("project", mcp.Required(), mcp.Description("Target project key, e.g. 'NEWPROJ'")),
)

var moveViaCloneTool = mcp.NewTool("move_via_clone",
	mcp.WithDescription("Single-shot move-by-clone: read source issue, create equivalent in target project, optionally copy comments, optionally link or delete the source. Use this when you lack the global Bulk Change permission and the public REST PUT does not support cross-project moves. Caveats: history, attachments, watchers, sprint membership, status, and custom fields are NOT preserved; the issue is renumbered (PLAT-123 → NEWPROJ-N)."),
	mcp.WithDestructiveHintAnnotation(true),
	mcp.WithString("key", mcp.Required(), mcp.Description("Source issue key, e.g. PROJ-123")),
	mcp.WithString("project", mcp.Required(), mcp.Description("Target project key, e.g. 'EDMOD'")),
	mcp.WithBoolean("copy_comments", mcp.Description("Copy comments from source to clone (default true)")),
	mcp.WithBoolean("delete_source", mcp.Description("Delete the source issue after a successful clone (default true). When false, link source as 'is cloned by' the new issue.")),
)
