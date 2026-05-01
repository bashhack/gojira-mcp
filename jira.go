// Package main implements gojira-mcp, an MCP server that wraps jira-cli.
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// issueKeyRe matches Jira issue keys like PROJ-123 or AB-1.
var issueKeyRe = regexp.MustCompile(`([A-Z][A-Z0-9]+-\d+)`)

// jiraRunner is the function used to execute jira-cli commands.
// It returns stdout, stderr, and any error. Tests replace this
// with a fake to avoid shelling out.
//
// Tests must not use t.Parallel() while swapping jiraRunner.
var jiraRunner = defaultRunJira

// defaultRunJira shells out to the jira binary with a 30-second timeout.
// The provided context is used for cancellation propagation.
func defaultRunJira(ctx context.Context, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	log.Printf("exec: jira %s [%d args]", args[0], len(args))

	cmd := exec.CommandContext(ctx, "jira", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// parseIssueKey extracts the first Jira issue key (e.g. "PROJ-123")
// from the given string. Returns an empty string if no key is found.
func parseIssueKey(output string) string {
	return issueKeyRe.FindString(output)
}

// isIssueKey reports whether s looks like a valid Jira issue key.
func isIssueKey(s string) bool {
	return s != "" && issueKeyRe.FindString(s) == s
}

// findActiveSprint queries jira-cli for the currently active sprint
// and returns its ID. When project is non-empty, the query is scoped
// to that project's board. Returns an error if no active sprint exists.
func findActiveSprint(ctx context.Context, project string) (string, error) {
	args := []string{"sprint", "list", "--state", "active", "--plain", "--no-headers", "--columns", "ID"}
	if project != "" {
		args = append(args, "-p", project)
	}
	out, errOut, err := jiraRunner(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("failed to list sprints: %s %s", errOut, err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "", fmt.Errorf("no active sprint found")
	}
	return strings.TrimSpace(lines[0]), nil
}
