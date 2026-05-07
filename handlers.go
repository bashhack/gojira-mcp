package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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
	for _, c := range req.GetStringSlice("components", nil) {
		args = append(args, "-C", c)
	}
	for _, v := range req.GetStringSlice("fix_version", nil) {
		args = append(args, "--fix-version", v)
	}
	for _, v := range req.GetStringSlice("affects_version", nil) {
		args = append(args, "--affects-version", v)
	}
	if est := req.GetString("original_estimate", ""); est != "" {
		args = append(args, "-e", est)
	}
	if reqArgs := req.GetArguments(); reqArgs != nil {
		if customRaw, ok := reqArgs["custom"]; ok {
			if customMap, ok := customRaw.(map[string]any); ok {
				for k, v := range customMap {
					args = append(args, "--custom", fmt.Sprintf("%s=%v", k, v))
				}
			}
		}
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
			id, err := findActiveSprint(ctx, project)
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
	body := req.GetString("body", "")
	priority := req.GetString("priority", "")
	assignee := req.GetString("assignee", "")
	epic := req.GetString("epic", "")
	labels := req.GetStringSlice("labels", nil)
	components := req.GetStringSlice("components", nil)
	fixVersions := req.GetStringSlice("fix_version", nil)
	affectsVersions := req.GetStringSlice("affects_version", nil)

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}

	args := []string{"issue", "edit", key, "--no-input"}
	hasFields := false

	if summary != "" {
		args = append(args, "-s", summary)
		hasFields = true
	}
	if body != "" {
		args = append(args, "-b", body)
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
	for _, c := range components {
		args = append(args, "-C", c)
		hasFields = true
	}
	for _, v := range fixVersions {
		args = append(args, "--fix-version", v)
		hasFields = true
	}
	for _, v := range affectsVersions {
		args = append(args, "--affects-version", v)
		hasFields = true
	}
	if reqArgs := req.GetArguments(); reqArgs != nil {
		if customRaw, ok := reqArgs["custom"]; ok {
			if customMap, ok := customRaw.(map[string]any); ok {
				for k, v := range customMap {
					args = append(args, "--custom", fmt.Sprintf("%s=%v", k, v))
					hasFields = true
				}
			}
		}
	}

	if !hasFields {
		return mcp.NewToolResultError("no fields to update — provide at least one of: summary, body, priority, assignee, epic, labels, components, fix_version, affects_version, custom"), nil
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

// handleAssignIssue assigns a Jira issue to a user, or unassigns it.
func handleAssignIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	assignee := req.GetString("assignee", "")

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}
	if assignee == "" {
		return mcp.NewToolResultError("assignee is required"), nil
	}

	args := []string{"issue", "assign", key, assignee}
	if assignee == "none" {
		args = []string{"issue", "assign", key, "x"}
	}

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to assign %s: %s\n%s", key, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleLinkIssues creates a link between two Jira issues.
func handleLinkIssues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	inward := req.GetString("inward", "")
	outward := req.GetString("outward", "")
	linkType := req.GetString("type", "")

	if inward == "" {
		return mcp.NewToolResultError("inward is required"), nil
	}
	if outward == "" {
		return mcp.NewToolResultError("outward is required"), nil
	}
	if linkType == "" {
		return mcp.NewToolResultError("type is required"), nil
	}

	out, errOut, err := jiraRunner(ctx, "issue", "link", inward, outward, linkType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to link %s to %s: %s\n%s", inward, outward, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleAddToSprint adds a Jira issue to the specified sprint.
// Pass "active" as the sprint value to auto-detect the active sprint.
func handleAddToSprint(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	sprint := req.GetString("sprint", "")
	project := req.GetString("project", "")

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}
	if sprint == "" {
		return mcp.NewToolResultError("sprint is required"), nil
	}

	sprintID := sprint
	if sprint == "active" {
		id, err := findActiveSprint(ctx, project)
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

// jiraUser represents a user returned by the Jira REST API.
type jiraUser struct {
	AccountID    string `json:"accountId"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

// handleSearchUsers searches for Jira users by name, email, or username.
func handleSearchUsers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	query := req.GetString("query", "")
	project := req.GetString("project", "")

	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	path := "/rest/api/3/user/assignable/search?query=" + url.QueryEscape(query)
	if project != "" {
		path += "&project=" + url.QueryEscape(project)
	}

	body, err := jiraAPIFetcher(ctx, "GET", path, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to search users: %s", err)), nil
	}

	var users []jiraUser
	if err := json.Unmarshal(body, &users); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse user search response: %s", err)), nil
	}

	if len(users) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No users found matching %q", query)), nil
	}

	var out strings.Builder
	for _, u := range users {
		if u.EmailAddress != "" {
			fmt.Fprintf(&out, "%s <%s> (accountId: %s)\n", u.DisplayName, u.EmailAddress, u.AccountID)
		} else {
			fmt.Fprintf(&out, "%s (accountId: %s)\n", u.DisplayName, u.AccountID)
		}
	}
	return mcp.NewToolResultText(strings.TrimSpace(out.String())), nil
}

// handleCloneIssue duplicates a Jira issue with optional field overrides.
func handleCloneIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}

	args := []string{"issue", "clone", key}
	if s := req.GetString("summary", ""); s != "" {
		args = append(args, "-s", s)
	}
	if p := req.GetString("priority", ""); p != "" {
		args = append(args, "-y", p)
	}
	if a := req.GetString("assignee", ""); a != "" {
		args = append(args, "-a", a)
	}
	for _, l := range req.GetStringSlice("labels", nil) {
		args = append(args, "-l", l)
	}
	for _, c := range req.GetStringSlice("components", nil) {
		args = append(args, "-C", c)
	}

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to clone %s: %s\n%s", key, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleDeleteIssue deletes a Jira issue, optionally cascading to subtasks.
func handleDeleteIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	cascade := req.GetBool("cascade", false)

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}

	args := []string{"issue", "delete", key}
	if cascade {
		args = append(args, "--cascade")
	}

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete %s: %s\n%s", key, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleUnlinkIssues removes a link between two Jira issues.
func handleUnlinkIssues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	inward := req.GetString("inward", "")
	outward := req.GetString("outward", "")

	if inward == "" {
		return mcp.NewToolResultError("inward is required"), nil
	}
	if outward == "" {
		return mcp.NewToolResultError("outward is required"), nil
	}

	out, errOut, err := jiraRunner(ctx, "issue", "unlink", inward, outward)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to unlink %s from %s: %s\n%s", inward, outward, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleWatchIssue adds a watcher to a Jira issue.
func handleWatchIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	watcher := req.GetString("watcher", "")

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}
	if watcher == "" {
		return mcp.NewToolResultError("watcher is required"), nil
	}

	out, errOut, err := jiraRunner(ctx, "issue", "watch", key, watcher)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add watcher to %s: %s\n%s", key, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleAddWorklog logs time spent on a Jira issue.
func handleAddWorklog(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	timeSpent := req.GetString("time_spent", "")

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}
	if timeSpent == "" {
		return mcp.NewToolResultError("time_spent is required"), nil
	}

	args := []string{"issue", "worklog", "add", key, timeSpent, "--no-input"}
	if c := req.GetString("comment", ""); c != "" {
		args = append(args, "--comment", c)
	}
	if s := req.GetString("started", ""); s != "" {
		args = append(args, "--started", s)
	}
	if e := req.GetString("new_estimate", ""); e != "" {
		args = append(args, "--new-estimate", e)
	}

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add worklog to %s: %s\n%s", key, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleCreateEpic creates a Jira epic.
func handleCreateEpic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	name := req.GetString("name", "")
	summary := req.GetString("summary", "")

	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	if summary == "" {
		return mcp.NewToolResultError("summary is required"), nil
	}

	args := []string{"epic", "create", "-n", name, "-s", summary, "--no-input"}
	if b := req.GetString("body", ""); b != "" {
		args = append(args, "-b", b)
	}
	if p := req.GetString("priority", ""); p != "" {
		args = append(args, "-y", p)
	}
	if a := req.GetString("assignee", ""); a != "" {
		args = append(args, "-a", a)
	}
	if proj := req.GetString("project", ""); proj != "" {
		args = append(args, "-p", proj)
	}
	for _, l := range req.GetStringSlice("labels", nil) {
		args = append(args, "-l", l)
	}
	for _, c := range req.GetStringSlice("components", nil) {
		args = append(args, "-C", c)
	}
	if reqArgs := req.GetArguments(); reqArgs != nil {
		if customRaw, ok := reqArgs["custom"]; ok {
			if customMap, ok := customRaw.(map[string]any); ok {
				for k, v := range customMap {
					args = append(args, "--custom", fmt.Sprintf("%s=%v", k, v))
				}
			}
		}
	}

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create epic: %s\n%s", errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleListEpics lists epics or issues within an epic.
func handleListEpics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	jql := req.GetString("jql", "")
	assignee := req.GetString("assignee", "")
	priority := req.GetString("priority", "")
	project := req.GetString("project", "")
	statuses := req.GetStringSlice("status", nil)
	labels := req.GetStringSlice("labels", nil)
	limit := req.GetInt("limit", 50)

	args := []string{"epic", "list"}
	if key != "" {
		args = append(args, key)
	}
	args = append(args, "--plain", "--no-truncate", "--columns", "TYPE,KEY,SUMMARY,STATUS,ASSIGNEE,PRIORITY")

	if jql != "" {
		args = append(args, "-q", jql)
	} else {
		if assignee != "" {
			args = append(args, "-a", assignee)
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
	}
	args = append(args, fmt.Sprintf("--paginate=%d", limit))

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list epics: %s\n%s", errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleAddIssuesToEpic bulk-adds issues to an epic (max 50).
func handleAddIssuesToEpic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	epic := req.GetString("epic", "")
	issues := req.GetStringSlice("issues", nil)

	if epic == "" {
		return mcp.NewToolResultError("epic is required"), nil
	}
	if len(issues) == 0 {
		return mcp.NewToolResultError("issues is required"), nil
	}

	args := []string{"epic", "add", epic}
	args = append(args, issues...)

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add issues to %s: %s\n%s", epic, errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleRemoveIssuesFromEpic removes the epic link from issues.
func handleRemoveIssuesFromEpic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	issues := req.GetStringSlice("issues", nil)

	if len(issues) == 0 {
		return mcp.NewToolResultError("issues is required"), nil
	}

	args := []string{"epic", "remove"}
	args = append(args, issues...)

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to remove issues from epic: %s\n%s", errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleListSprints lists sprints in the project board.
func handleListSprints(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	state := req.GetString("state", "")
	project := req.GetString("project", "")

	args := []string{"sprint", "list", "--table", "--plain", "--no-headers", "--columns", "ID,NAME,START,END,STATE"}
	if state != "" {
		args = append(args, "--state", state)
	}
	if project != "" {
		args = append(args, "-p", project)
	}

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list sprints: %s\n%s", errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleListBoards lists Jira boards.
func handleListBoards(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	project := req.GetString("project", "")

	args := []string{"board", "list", "--plain", "--no-headers"}
	if project != "" {
		args = append(args, "-p", project)
	}

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list boards: %s\n%s", errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleListProjects lists Jira projects.
func handleListProjects(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	args := []string{"project", "list", "--plain", "--no-headers"}

	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list projects: %s\n%s", errOut, out)), nil
	}
	return mcp.NewToolResultText(strings.TrimSpace(out)), nil
}

// handleChangeIssueType changes the issue type of an existing Jira issue.
// jira-cli does not expose this operation, so the request is sent directly
// to the Jira REST API as PUT /rest/api/3/issue/{key} with a JSON body
// of {"fields":{"issuetype":{"name":<type>}}}.
func handleChangeIssueType(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	issueType := req.GetString("type", "")

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}
	if issueType == "" {
		return mcp.NewToolResultError("type is required"), nil
	}

	body, err := json.Marshal(map[string]any{
		"fields": map[string]any{
			"issuetype": map[string]string{"name": issueType},
		},
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal request body: %s", err)), nil
	}

	if _, err := jiraAPIFetcher(ctx, "PUT", "/rest/api/3/issue/"+url.PathEscape(key), body); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to change %s issue type to %q: %s", key, issueType, err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("✓ %s issue type changed to %q", key, issueType)), nil
}

// handleMoveViaClone moves an issue to another project by clone-and-delete.
// Used as a fallback when the global Bulk Change permission is unavailable
// and the REST PUT endpoint silently no-ops on project-field changes.
//
// The flow is six steps; on partial failure the new key (if any) is reported
// so the caller can decide whether to retry or clean up:
//
//  1. GET source issue fields (summary, description, issue type, priority,
//     labels, components)
//  2. Build the create payload, copying ADF description as-is
//  3. POST to /issue, capturing the new key
//  4. If copy_comments: GET source comments, POST each to the clone
//  5. If !delete_source: POST a Cloners link from source to clone
//  6. If delete_source: DELETE the source issue
//
// History, attachments, watchers, sprint membership, status, and custom
// fields are NOT preserved.
func handleMoveViaClone(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:gocritic // hugeParam: signature required by mcp-go ToolHandlerFunc
	key := req.GetString("key", "")
	project := req.GetString("project", "")
	copyComments := req.GetBool("copy_comments", true)
	deleteSource := req.GetBool("delete_source", true)

	if key == "" {
		return mcp.NewToolResultError("key is required"), nil
	}
	if project == "" {
		return mcp.NewToolResultError("project is required"), nil
	}

	// Step 1: GET source fields
	sourceBody, err := jiraAPIFetcher(ctx, "GET", "/rest/api/3/issue/"+url.PathEscape(key)+"?fields=summary,description,issuetype,priority,labels,components", nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("step 1 (fetch source %s): %s", key, err)), nil
	}
	var source struct {
		Fields struct {
			Summary     string          `json:"summary"`
			Description json.RawMessage `json:"description"`
			IssueType   struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Priority *struct {
				Name string `json:"name"`
			} `json:"priority"`
			Labels     []string `json:"labels"`
			Components []struct {
				Name string `json:"name"`
			} `json:"components"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(sourceBody, &source); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("step 1 (parse source): %s", err)), nil
	}

	// Step 2: build create payload
	fields := map[string]any{
		"project":   map[string]string{"key": project},
		"summary":   source.Fields.Summary,
		"issuetype": map[string]string{"name": source.Fields.IssueType.Name},
	}
	if len(source.Fields.Description) > 0 && string(source.Fields.Description) != "null" {
		fields["description"] = source.Fields.Description
	}
	if source.Fields.Priority != nil && source.Fields.Priority.Name != "" {
		fields["priority"] = map[string]string{"name": source.Fields.Priority.Name}
	}
	if len(source.Fields.Labels) > 0 {
		fields["labels"] = source.Fields.Labels
	}
	if len(source.Fields.Components) > 0 {
		comps := make([]map[string]string, len(source.Fields.Components))
		for i, c := range source.Fields.Components {
			comps[i] = map[string]string{"name": c.Name}
		}
		fields["components"] = comps
	}
	createBody, err := json.Marshal(map[string]any{"fields": fields})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("step 2 (marshal create payload): %s", err)), nil
	}

	// Step 3: POST new issue
	createResp, err := jiraAPIFetcher(ctx, "POST", "/rest/api/3/issue", createBody)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("step 3 (create in %s): %s", project, err)), nil
	}
	var created struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(createResp, &created); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("step 3 (parse create response): %s", err)), nil
	}
	if created.Key == "" {
		return mcp.NewToolResultError(fmt.Sprintf("step 3: create returned no key — body: %s", string(createResp))), nil
	}

	commentsCopied := 0
	linkAdded := false
	sourceDeleted := false

	// Used after step 3 — always include the new key in errors so the caller
	// can clean up or follow up.
	partial := func(step string, e error) *mcp.CallToolResult {
		return mcp.NewToolResultError(fmt.Sprintf("Partial success — clone exists at %s but %s failed: %s\nState: %s", created.Key, step, e, summarizeMoveViaClone(key, created.Key, project, commentsCopied, linkAdded, sourceDeleted)))
	}

	// Step 4: copy comments
	if copyComments {
		commentsResp, err := jiraAPIFetcher(ctx, "GET", "/rest/api/3/issue/"+url.PathEscape(key)+"/comment", nil)
		if err != nil {
			return partial("step 4 (fetch source comments)", err), nil
		}
		var comments struct {
			Comments []struct {
				Body json.RawMessage `json:"body"`
			} `json:"comments"`
		}
		if err := json.Unmarshal(commentsResp, &comments); err != nil {
			return partial("step 4 (parse comments)", err), nil
		}
		for i, c := range comments.Comments {
			if len(c.Body) == 0 || string(c.Body) == "null" {
				continue
			}
			cb, err := json.Marshal(map[string]any{"body": c.Body})
			if err != nil {
				return partial(fmt.Sprintf("step 4 (marshal comment %d/%d)", i+1, len(comments.Comments)), err), nil
			}
			if _, err := jiraAPIFetcher(ctx, "POST", "/rest/api/3/issue/"+url.PathEscape(created.Key)+"/comment", cb); err != nil {
				return partial(fmt.Sprintf("step 4 (post comment %d/%d)", i+1, len(comments.Comments)), err), nil
			}
			commentsCopied++
		}
	}

	// Step 5: link (only if keeping source)
	if !deleteSource {
		linkBody, err := json.Marshal(map[string]any{
			"type":         map[string]string{"name": "Cloners"},
			"inwardIssue":  map[string]string{"key": key},
			"outwardIssue": map[string]string{"key": created.Key},
		})
		if err != nil {
			return partial("step 5 (marshal link)", err), nil
		}
		if _, err := jiraAPIFetcher(ctx, "POST", "/rest/api/3/issueLink", linkBody); err != nil {
			return partial("step 5 (add Cloners link)", err), nil
		}
		linkAdded = true
	}

	// Step 6: delete source
	if deleteSource {
		if _, err := jiraAPIFetcher(ctx, "DELETE", "/rest/api/3/issue/"+url.PathEscape(key), nil); err != nil {
			return partial("step 6 (delete source)", err), nil
		}
		sourceDeleted = true
	}

	return mcp.NewToolResultText(summarizeMoveViaClone(key, created.Key, project, commentsCopied, linkAdded, sourceDeleted)), nil
}

func summarizeMoveViaClone(sourceKey, newKey, project string, commentsCopied int, linkAdded, sourceDeleted bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "✓ Cloned %s → %s in project %s", sourceKey, newKey, project)
	if commentsCopied > 0 {
		fmt.Fprintf(&b, "\n  comments copied: %d", commentsCopied)
	}
	if linkAdded {
		fmt.Fprintf(&b, "\n  link added: %s ← Cloners ← %s", sourceKey, newKey)
	}
	if sourceDeleted {
		fmt.Fprintf(&b, "\n  source %s deleted", sourceKey)
	}
	return b.String()
}
