package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// handleCreateIssue creates a Jira issue and optionally assigns it to a user,
// links it to an epic, transitions its status, and adds it to a sprint.
// Post-creation steps are best-effort: failures are reported in the result
// text but do not prevent the overall operation from succeeding.
func handleCreateIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	summary := req.GetString("summary", "")
	issueType := req.GetString("type", "")
	description := req.GetString("description", "")
	priority := req.GetString("priority", "")
	assignee := req.GetString("assignee", "")
	epic := req.GetString("epic", "")
	status := req.GetString("status", "")
	sprint := req.GetString("sprint", "")
	project := req.GetString("project", "")
	labels := req.GetStringSlice("labels", nil)

	if summary == "" {
		return mcp.NewToolResultError("summary is required"), nil
	}
	if issueType == "" {
		return mcp.NewToolResultError("type is required"), nil
	}

	args := []string{"issue", "create", "-t", issueType, "-s", summary, "--no-input"}

	if description != "" {
		args = append(args, "-b", description)
	}
	if priority != "" {
		args = append(args, "-y", priority)
	}
	if assignee != "" {
		args = append(args, "-a", assignee)
	}
	if epic != "" {
		args = append(args, "-P", epic)
	}
	if project != "" {
		args = append(args, "-p", project)
	}
	for _, l := range labels {
		args = append(args, "-l", l)
	}

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create issue: %s\n%s", errOut, out)), nil
	}

	key := parseIssueKey(out)
	if key == "" {
		return mcp.NewToolResultError(fmt.Sprintf("Issue created but could not parse key from output:\n%s", out)), nil
	}

	var report strings.Builder
	report.WriteString(fmt.Sprintf("Created %s: %q\n", key, summary))

	if status != "" {
		_, errOut, err := jiraRunner(ctx, "issue", "move", key, status)
		if err != nil {
			report.WriteString(fmt.Sprintf("  Status: %s -- FAILED: %s\n", status, strings.TrimSpace(errOut)))
		} else {
			report.WriteString(fmt.Sprintf("  Status: %s -- OK\n", status))
		}
	}

	if sprint != "" {
		sprintID := sprint
		if sprint == "active" {
			id, err := findActiveSprint(ctx)
			if err != nil {
				report.WriteString(fmt.Sprintf("  Sprint: active -- FAILED: %s\n", err))
				sprintID = ""
			} else {
				sprintID = id
			}
		}
		if sprintID != "" {
			_, errOut, err := jiraRunner(ctx, "sprint", "add", sprintID, key)
			if err != nil {
				report.WriteString(fmt.Sprintf("  Sprint: %s -- FAILED: %s\n", sprintID, strings.TrimSpace(errOut)))
			} else {
				label := sprintID
				if sprint == "active" {
					label = sprintID + " (active)"
				}
				report.WriteString(fmt.Sprintf("  Sprint: %s -- OK\n", label))
			}
		}
	}

	for line := range strings.SplitSeq(out, "\n") {
		if strings.Contains(line, "atlassian.net/browse/") {
			report.WriteString(strings.TrimSpace(line) + "\n")
			break
		}
	}

	return mcp.NewToolResultText(report.String()), nil
}

// handleEditIssue updates fields on an existing Jira issue.
func handleEditIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	summary := req.GetString("summary", "")
	priority := req.GetString("priority", "")
	assignee := req.GetString("assignee", "")
	epic := req.GetString("epic", "")
	labels := req.GetStringSlice("labels", nil)

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}

	args := []string{"issue", "edit", key, "--no-input"}
	hasFields := false

	if summary != "" {
		args = append(args, "-s", summary)
		hasFields = true
	}
	if priority != "" {
		args = append(args, "-y", priority)
		hasFields = true
	}
	if assignee != "" {
		args = append(args, "-a", assignee)
		hasFields = true
	}
	if epic != "" {
		args = append(args, "-P", epic)
		hasFields = true
	}
	for _, l := range labels {
		args = append(args, "-l", l)
		hasFields = true
	}

	if !hasFields {
		return mcp.NewToolResultError("no fields to update — provide at least one of: summary, priority, assignee, epic, labels"), nil
	}

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to edit %s: %s\n%s", key, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleMoveIssue transitions a Jira issue to a new status.
func handleMoveIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	status := req.GetString("status", "")

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}
	if status == "" {
		return mcp.NewToolResultError("status is required"), nil
	}

	out, errOut, err := jiraRunner(ctx, "issue", "move", key, status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to move %s to %q: %s\n%s", key, status, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleViewIssue returns the details of a Jira issue in plain text.
func handleViewIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	comments := req.GetInt("comments", 5)

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}

	args := []string{"issue", "view", key, "--plain", "--comments", fmt.Sprintf("%d", comments)}

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to view %s: %s\n%s", key, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleListIssues lists Jira issues matching the given filters or JQL query.
func handleListIssues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	jql := req.GetString("jql", "")
	assignee := req.GetString("assignee", "")
	issueType := req.GetString("type", "")
	priority := req.GetString("priority", "")
	parent := req.GetString("parent", "")
	project := req.GetString("project", "")
	statuses := req.GetStringSlice("status", nil)
	labels := req.GetStringSlice("labels", nil)
	limit := req.GetInt("limit", 20)

	args := []string{"issue", "list", "--plain", "--no-truncate", "--columns", "TYPE,KEY,SUMMARY,STATUS,ASSIGNEE,PRIORITY"}

	if jql != "" {
		args = append(args, "-q", jql)
	} else {
		if assignee != "" {
			args = append(args, "-a", assignee)
		}
		if issueType != "" {
			args = append(args, "-t", issueType)
		}
		if priority != "" {
			args = append(args, "-y", priority)
		}
		if project != "" {
			args = append(args, "-p", project)
		}
		for _, s := range statuses {
			args = append(args, "-s", s)
		}
		for _, l := range labels {
			args = append(args, "-l", l)
		}
		if parent != "" {
			if !issueKeyRe.MatchString(parent) {
				return mcp.NewToolResultError(fmt.Sprintf("invalid parent key %q — must be a Jira issue key like PROJ-123", parent)), nil
			}
			args = append(args, "-q", fmt.Sprintf("parent = %s", parent))
		}
	}

	args = append(args, fmt.Sprintf("--paginate=%d", limit))

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list issues: %s\n%s", errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleAddComment adds a comment to a Jira issue.
func handleAddComment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	body := req.GetString("body", "")

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}
	if body == "" {
		return mcp.NewToolResultError("body is required"), nil
	}

	out, errOut, err := jiraRunner(ctx, "issue", "comment", "add", key, body, "--no-input")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add comment to %s: %s\n%s", key, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleAddToSprint adds a Jira issue to the specified sprint.
// Pass "active" as the sprint value to auto-detect the active sprint.
func handleAddToSprint(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	sprint := req.GetString("sprint", "")

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}
	if sprint == "" {
		return mcp.NewToolResultError("sprint is required"), nil
	}

	sprintID := sprint
	if sprint == "active" {
		id, err := findActiveSprint(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to find active sprint: %s", err)), nil
		}
		sprintID = id
	}

	out, errOut, err := jiraRunner(ctx, "sprint", "add", sprintID, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add %s to sprint %s: %s\n%s", key, sprintID, errOut, out)), nil
	}

	label := sprintID
	if sprint == "active" {
		label = sprintID + " (active)"
	}
	return mcp.NewToolResultText(fmt.Sprintf("Added %s to sprint %s", key, label)), nil
}
