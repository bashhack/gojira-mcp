package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

var errFake = errors.New("fake error")

func fakeRunner(stdout, stderr string, err error) func(context.Context, ...string) (string, string, error) {
	return func(_ context.Context, _ ...string) (string, string, error) {
		return stdout, stderr, err
	}
}

type fakeResponse struct {
	err    error
	stdout string
	stderr string
}

type callRecord struct {
	args []string
}

type sequenceRunner struct {
	calls     []callRecord
	responses []fakeResponse
}

func (s *sequenceRunner) run(_ context.Context, args ...string) (string, string, error) {
	s.calls = append(s.calls, callRecord{args: args})
	idx := len(s.calls) - 1
	if idx >= len(s.responses) {
		idx = len(s.responses) - 1
	}
	r := s.responses[idx]
	return r.stdout, r.stderr, r.err
}

func makeCallToolRequest(t *testing.T, params map[string]any) mcp.CallToolRequest {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}
	var req mcp.CallToolRequest
	req.Params.Name = "test"
	if err := json.Unmarshal(raw, &req.Params.Arguments); err != nil {
		t.Fatalf("failed to unmarshal params: %v", err)
	}
	return req
}

func TestHandleCreateIssue(t *testing.T) {
	tests := map[string]struct {
		runner     *sequenceRunner
		params     map[string]any
		wantInText []string
		wantErr    bool
	}{
		"basic create": {
			params: map[string]any{
				"summary": "Fix login bug",
				"type":    "Bug",
			},
			runner: &sequenceRunner{
				responses: []fakeResponse{
					{stdout: "✓ Issue created\nhttps://example.atlassian.net/browse/TEST-100\n"},
				},
			},
			wantInText: []string{"Created TEST-100", "Fix login bug"},
		},
		"create with all options": {
			params: map[string]any{
				"summary":  "Add caching",
				"type":     "Story",
				"priority": "High",
				"assignee": "alice@example.com",
				"epic":     "TEST-50",
				"status":   "In Progress",
				"sprint":   "42",
				"labels":   []any{"backend", "perf"},
			},
			runner: &sequenceRunner{
				responses: []fakeResponse{
					{stdout: "https://example.atlassian.net/browse/TEST-200\n"},
					{stdout: "moved"},
					{stdout: "sprint ok"},
				},
			},
			wantInText: []string{
				"Created TEST-200",
				"Status: In Progress -- OK",
				"Sprint: 42 -- OK",
			},
		},
		"create with active sprint": {
			params: map[string]any{
				"summary": "Sprint test",
				"type":    "Task",
				"sprint":  "active",
			},
			runner: &sequenceRunner{
				responses: []fakeResponse{
					{stdout: "https://example.atlassian.net/browse/TEST-300\n"},
					{stdout: "3601\n"},
					{stdout: "added"},
				},
			},
			wantInText: []string{"Created TEST-300", "Sprint: 3601 (active) -- OK"},
		},
		"create fails": {
			params: map[string]any{
				"summary": "Fail",
				"type":    "Task",
			},
			runner: &sequenceRunner{
				responses: []fakeResponse{
					{stderr: "auth error", err: errFake},
				},
			},
			wantErr:    true,
			wantInText: []string{"Failed to create issue"},
		},
		"create succeeds but status transition fails": {
			params: map[string]any{
				"summary": "Partial",
				"type":    "Task",
				"status":  "In Review",
			},
			runner: &sequenceRunner{
				responses: []fakeResponse{
					{stdout: "https://example.atlassian.net/browse/TEST-400\n"},
					{stderr: "transition not available", err: errFake},
				},
			},
			wantInText: []string{"Created TEST-400", "Status: In Review -- FAILED"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			jiraRunner = tc.runner.run
			t.Cleanup(func() { jiraRunner = defaultRunJira })

			result, err := handleCreateIssue(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}

			text := resultText(t, result)
			if tc.wantErr && !result.IsError {
				t.Fatal("expected tool error, got success")
			}
			if !tc.wantErr && result.IsError {
				t.Fatalf("expected success, got tool error: %s", text)
			}
			for _, want := range tc.wantInText {
				if !strings.Contains(text, want) {
					t.Errorf("result missing %q\nfull text: %s", want, text)
				}
			}
		})
	}
}

func TestHandleCreateIssueBuildArgs(t *testing.T) {
	runner := &sequenceRunner{
		responses: []fakeResponse{
			{stdout: "https://example.atlassian.net/browse/TEST-1\n"},
		},
	}
	jiraRunner = runner.run
	t.Cleanup(func() { jiraRunner = defaultRunJira })

	req := makeCallToolRequest(t, map[string]any{
		"summary":     "Test args",
		"type":        "Story",
		"description": "A long\nmultiline\ndescription",
		"priority":    "High",
		"assignee":    "bob@example.com",
		"epic":        "TEST-1",
		"project":     "TEST",
		"labels":      []any{"sec", "api"},
		"custom":      map[string]any{"story-points": "5"},
	})

	if _, err := handleCreateIssue(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.calls) == 0 {
		t.Fatal("no calls recorded")
	}
	args := strings.Join(runner.calls[0].args, " ")

	for _, want := range []string{"-t Story", "-s Test args", "-b A long\nmultiline\ndescription", "-y High", "-a bob@example.com", "-P TEST-1", "-p TEST", "-l sec", "-l api", "--custom story-points=5"} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q\nfull args: %s", want, args)
		}
	}
}

func TestHandleCreateIssueMissingRequired(t *testing.T) {
	tests := map[string]struct {
		params     map[string]any
		wantInText string
	}{
		"missing summary": {
			params:     map[string]any{"type": "Bug"},
			wantInText: "summary is required",
		},
		"missing type": {
			params:     map[string]any{"summary": "Fix it"},
			wantInText: "type is required",
		},
		"both missing": {
			params:     map[string]any{},
			wantInText: "summary is required",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := handleCreateIssue(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Fatal("expected error result")
			}
			text := resultText(t, result)
			if !strings.Contains(text, tc.wantInText) {
				t.Errorf("result missing %q\nfull: %s", tc.wantInText, text)
			}
		})
	}
}

func TestHandleEditIssue(t *testing.T) {
	tests := map[string]struct {
		err        error
		params     map[string]any
		stdout     string
		stderr     string
		wantInText string
		wantErr    bool
	}{
		"edit succeeds": {
			params:     map[string]any{"key": "TEST-1", "summary": "New title", "priority": "Low"},
			stdout:     "✓ Issue updated\nhttps://example.atlassian.net/browse/TEST-1\n",
			wantInText: "Issue updated",
		},
		"edit fails": {
			params:     map[string]any{"key": "TEST-1", "summary": "x"},
			stderr:     "not found",
			err:        errFake,
			wantErr:    true,
			wantInText: "Failed to edit",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			jiraRunner = fakeRunner(tc.stdout, tc.stderr, tc.err)
			t.Cleanup(func() { jiraRunner = defaultRunJira })

			result, err := handleEditIssue(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}

			text := resultText(t, result)
			if tc.wantErr && !result.IsError {
				t.Fatal("expected error result")
			}
			if !strings.Contains(text, tc.wantInText) {
				t.Errorf("result missing %q\nfull: %s", tc.wantInText, text)
			}
		})
	}
}

func TestHandleEditIssueCustomFields(t *testing.T) {
	runner := &sequenceRunner{
		responses: []fakeResponse{
			{stdout: "✓ Issue updated\nhttps://example.atlassian.net/browse/TEST-1\n"},
		},
	}
	jiraRunner = runner.run
	t.Cleanup(func() { jiraRunner = defaultRunJira })

	req := makeCallToolRequest(t, map[string]any{
		"key":    "TEST-1",
		"custom": map[string]any{"story-points": "3"},
	})

	result, err := handleEditIssue(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	if len(runner.calls) == 0 {
		t.Fatal("no calls recorded")
	}
	args := strings.Join(runner.calls[0].args, " ")
	if !strings.Contains(args, "--custom story-points=3") {
		t.Errorf("args missing custom field\nfull args: %s", args)
	}
}

func TestHandleEditIssueNoFields(t *testing.T) {
	result, err := handleEditIssue(context.Background(), makeCallToolRequest(t, map[string]any{"key": "TEST-1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for no fields")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "no fields to update") {
		t.Errorf("unexpected error: %s", text)
	}
}

func TestHandleEditIssueMissingKey(t *testing.T) {
	result, err := handleEditIssue(context.Background(), makeCallToolRequest(t, map[string]any{"summary": "x"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "key is required") {
		t.Errorf("unexpected error: %s", text)
	}
}

func TestHandleMoveIssueMissingParams(t *testing.T) {
	tests := map[string]struct {
		params     map[string]any
		wantInText string
	}{
		"missing key": {
			params:     map[string]any{"status": "Done"},
			wantInText: "key is required",
		},
		"missing status": {
			params:     map[string]any{"key": "TEST-1"},
			wantInText: "status is required",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := handleMoveIssue(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Fatal("expected error result")
			}
			text := resultText(t, result)
			if !strings.Contains(text, tc.wantInText) {
				t.Errorf("result missing %q\nfull: %s", tc.wantInText, text)
			}
		})
	}
}

func TestHandleMoveIssue(t *testing.T) {
	tests := map[string]struct {
		err        error
		params     map[string]any
		stdout     string
		stderr     string
		wantInText string
		wantErr    bool
	}{
		"move succeeds": {
			params:     map[string]any{"key": "TEST-1", "status": "Done"},
			stdout:     "✓ Issue transitioned",
			wantInText: "Issue transitioned",
		},
		"move fails": {
			params:     map[string]any{"key": "TEST-1", "status": "Invalid"},
			stderr:     "transition not found",
			err:        errFake,
			wantErr:    true,
			wantInText: "Failed to move",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			jiraRunner = fakeRunner(tc.stdout, tc.stderr, tc.err)
			t.Cleanup(func() { jiraRunner = defaultRunJira })

			result, err := handleMoveIssue(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(t, result)
			if tc.wantErr && !result.IsError {
				t.Fatal("expected error result")
			}
			if !strings.Contains(text, tc.wantInText) {
				t.Errorf("result missing %q\nfull: %s", tc.wantInText, text)
			}
		})
	}
}

func TestHandleViewIssueMissingKey(t *testing.T) {
	result, err := handleViewIssue(context.Background(), makeCallToolRequest(t, map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "key is required") {
		t.Errorf("unexpected error: %s", text)
	}
}

func TestHandleViewIssueError(t *testing.T) {
	jiraRunner = fakeRunner("", "not found", errFake)
	t.Cleanup(func() { jiraRunner = defaultRunJira })

	result, err := handleViewIssue(context.Background(), makeCallToolRequest(t, map[string]any{"key": "TEST-999"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "Failed to view") {
		t.Errorf("unexpected error: %s", text)
	}
}

func TestHandleViewIssue(t *testing.T) {
	jiraRunner = fakeRunner("TYPE: Bug\nKEY: TEST-1\nSUMMARY: Fix it\n", "", nil)
	t.Cleanup(func() { jiraRunner = defaultRunJira })

	result, err := handleViewIssue(context.Background(), makeCallToolRequest(t, map[string]any{"key": "TEST-1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "TEST-1") {
		t.Errorf("expected TEST-1 in output, got: %s", text)
	}
}

func TestHandleListIssues(t *testing.T) {
	tests := map[string]struct {
		params     map[string]any
		wantInArgs []string
	}{
		"with jql": {
			params:     map[string]any{"jql": "project = TEST ORDER BY created"},
			wantInArgs: []string{"-q", "project = TEST ORDER BY created"},
		},
		"with filters": {
			params:     map[string]any{"assignee": "alice", "type": "Bug", "priority": "High"},
			wantInArgs: []string{"-a", "alice", "-t", "Bug", "-y", "High"},
		},
		"with parent": {
			params:     map[string]any{"parent": "TEST-100"},
			wantInArgs: []string{"-q", "parent = TEST-100"},
		},
		"custom limit": {
			params:     map[string]any{"limit": float64(50)},
			wantInArgs: []string{"--paginate=50"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runner := &sequenceRunner{
				responses: []fakeResponse{
					{stdout: "TYPE\tKEY\tSUMMARY\n"},
				},
			}
			jiraRunner = runner.run
			t.Cleanup(func() { jiraRunner = defaultRunJira })

			if _, err := handleListIssues(context.Background(), makeCallToolRequest(t, tc.params)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(runner.calls) == 0 {
				t.Fatal("no calls recorded")
			}
			args := strings.Join(runner.calls[0].args, " ")
			for _, want := range tc.wantInArgs {
				if !strings.Contains(args, want) {
					t.Errorf("args missing %q\nfull args: %s", want, args)
				}
			}
		})
	}
}

func TestHandleListIssuesInvalidParent(t *testing.T) {
	result, err := handleListIssues(context.Background(), makeCallToolRequest(t, map[string]any{
		"parent": "not-a-key",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid parent key")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "invalid parent key") {
		t.Errorf("unexpected error: %s", text)
	}
}

func TestHandleAddCommentMissingParams(t *testing.T) {
	tests := map[string]struct {
		params     map[string]any
		wantInText string
	}{
		"missing key": {
			params:     map[string]any{"body": "hello"},
			wantInText: "key is required",
		},
		"missing body": {
			params:     map[string]any{"key": "TEST-1"},
			wantInText: "body is required",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := handleAddComment(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Fatal("expected error result")
			}
			text := resultText(t, result)
			if !strings.Contains(text, tc.wantInText) {
				t.Errorf("result missing %q\nfull: %s", tc.wantInText, text)
			}
		})
	}
}

func TestHandleAddCommentError(t *testing.T) {
	jiraRunner = fakeRunner("", "failed", errFake)
	t.Cleanup(func() { jiraRunner = defaultRunJira })

	result, err := handleAddComment(context.Background(), makeCallToolRequest(t, map[string]any{"key": "TEST-1", "body": "hi"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "Failed to add comment") {
		t.Errorf("unexpected error: %s", text)
	}
}

func TestHandleAddComment(t *testing.T) {
	jiraRunner = fakeRunner("✓ Comment added", "", nil)
	t.Cleanup(func() { jiraRunner = defaultRunJira })

	result, err := handleAddComment(context.Background(), makeCallToolRequest(t, map[string]any{
		"key":  "TEST-1",
		"body": "Looks good!",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "Comment added") {
		t.Errorf("unexpected result: %s", text)
	}
}

func TestHandleAddToSprintMissingParams(t *testing.T) {
	tests := map[string]struct {
		params     map[string]any
		wantInText string
	}{
		"missing key": {
			params:     map[string]any{"sprint": "42"},
			wantInText: "key is required",
		},
		"missing sprint": {
			params:     map[string]any{"key": "TEST-1"},
			wantInText: "sprint is required",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := handleAddToSprint(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Fatal("expected error result")
			}
			text := resultText(t, result)
			if !strings.Contains(text, tc.wantInText) {
				t.Errorf("result missing %q\nfull: %s", tc.wantInText, text)
			}
		})
	}
}

func TestHandleAddToSprint(t *testing.T) {
	tests := map[string]struct {
		runner     *sequenceRunner
		params     map[string]any
		wantInText string
		wantErr    bool
	}{
		"explicit sprint ID": {
			params: map[string]any{"key": "TEST-1", "sprint": "42"},
			runner: &sequenceRunner{
				responses: []fakeResponse{
					{stdout: "added"},
				},
			},
			wantInText: "Added TEST-1 to sprint 42",
		},
		"active sprint": {
			params: map[string]any{"key": "TEST-1", "sprint": "active"},
			runner: &sequenceRunner{
				responses: []fakeResponse{
					{stdout: "3601\n"},
					{stdout: "added"},
				},
			},
			wantInText: "sprint 3601 (active)",
		},
		"no active sprint found": {
			params: map[string]any{"key": "TEST-1", "sprint": "active"},
			runner: &sequenceRunner{
				responses: []fakeResponse{
					{stdout: "\n"},
				},
			},
			wantErr:    true,
			wantInText: "Failed to find active sprint",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			jiraRunner = tc.runner.run
			t.Cleanup(func() { jiraRunner = defaultRunJira })

			result, err := handleAddToSprint(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			text := resultText(t, result)
			if tc.wantErr && !result.IsError {
				t.Fatal("expected error result")
			}
			if !strings.Contains(text, tc.wantInText) {
				t.Errorf("result missing %q\nfull: %s", tc.wantInText, text)
			}
		})
	}
}

func TestHandleAssignIssue(t *testing.T) {
	tests := map[string]struct {
		err        error
		params     map[string]any
		stdout     string
		stderr     string
		wantInText string
		wantErr    bool
	}{
		"assign to user": {
			params:     map[string]any{"key": "TEST-1", "assignee": "alice@example.com"},
			stdout:     "✓ Issue assigned",
			wantInText: "Issue assigned",
		},
		"unassign": {
			params:     map[string]any{"key": "TEST-1", "assignee": "none"},
			stdout:     "✓ Issue unassigned",
			wantInText: "Issue unassigned",
		},
		"assign fails": {
			params:     map[string]any{"key": "TEST-1", "assignee": "nobody@example.com"},
			stderr:     "user not found",
			err:        errFake,
			wantErr:    true,
			wantInText: "Failed to assign",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			jiraRunner = fakeRunner(tc.stdout, tc.stderr, tc.err)
			t.Cleanup(func() { jiraRunner = defaultRunJira })

			result, err := handleAssignIssue(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			text := resultText(t, result)
			if tc.wantErr && !result.IsError {
				t.Fatal("expected error result")
			}
			if !strings.Contains(text, tc.wantInText) {
				t.Errorf("result missing %q\nfull: %s", tc.wantInText, text)
			}
		})
	}
}

func TestHandleAssignIssueMissingParams(t *testing.T) {
	tests := map[string]struct {
		params     map[string]any
		wantInText string
	}{
		"missing key": {
			params:     map[string]any{"assignee": "alice@example.com"},
			wantInText: "key is required",
		},
		"missing assignee": {
			params:     map[string]any{"key": "TEST-1"},
			wantInText: "assignee is required",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := handleAssignIssue(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Fatal("expected error result")
			}
			text := resultText(t, result)
			if !strings.Contains(text, tc.wantInText) {
				t.Errorf("result missing %q\nfull: %s", tc.wantInText, text)
			}
		})
	}
}

func TestHandleLinkIssues(t *testing.T) {
	jiraRunner = fakeRunner("✓ Issues linked", "", nil)
	t.Cleanup(func() { jiraRunner = defaultRunJira })

	result, err := handleLinkIssues(context.Background(), makeCallToolRequest(t, map[string]any{
		"inward":  "TEST-1",
		"outward": "TEST-2",
		"type":    "Blocks",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "Issues linked") {
		t.Errorf("unexpected result: %s", text)
	}
}

func TestHandleLinkIssuesError(t *testing.T) {
	jiraRunner = fakeRunner("", "link type not found", errFake)
	t.Cleanup(func() { jiraRunner = defaultRunJira })

	result, err := handleLinkIssues(context.Background(), makeCallToolRequest(t, map[string]any{
		"inward":  "TEST-1",
		"outward": "TEST-2",
		"type":    "InvalidType",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "Failed to link") {
		t.Errorf("unexpected error: %s", text)
	}
}

func TestHandleLinkIssuesMissingParams(t *testing.T) {
	tests := map[string]struct {
		params     map[string]any
		wantInText string
	}{
		"missing inward": {
			params:     map[string]any{"outward": "TEST-2", "type": "Blocks"},
			wantInText: "inward is required",
		},
		"missing outward": {
			params:     map[string]any{"inward": "TEST-1", "type": "Blocks"},
			wantInText: "outward is required",
		},
		"missing type": {
			params:     map[string]any{"inward": "TEST-1", "outward": "TEST-2"},
			wantInText: "type is required",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := handleLinkIssues(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Fatal("expected error result")
			}
			text := resultText(t, result)
			if !strings.Contains(text, tc.wantInText) {
				t.Errorf("result missing %q\nfull: %s", tc.wantInText, text)
			}
		})
	}
}

func fakeAPIFetcher(body []byte, err error) func(context.Context, string, string) ([]byte, error) {
	return func(_ context.Context, _, _ string) ([]byte, error) {
		return body, err
	}
}

func TestHandleSearchUsers(t *testing.T) {
	tests := map[string]struct {
		fetcher    func(context.Context, string, string) ([]byte, error)
		params     map[string]any
		wantInText []string
		wantErr    bool
	}{
		"basic search": {
			params: map[string]any{"query": "alice"},
			fetcher: fakeAPIFetcher([]byte(`[
				{"accountId":"abc123","displayName":"Alice Smith","emailAddress":"alice@example.com"},
				{"accountId":"def456","displayName":"Alice Jones","emailAddress":"alice.jones@example.com"}
			]`), nil),
			wantInText: []string{"Alice Smith", "alice@example.com", "Alice Jones"},
		},
		"no results": {
			params:     map[string]any{"query": "zzzzzzz"},
			fetcher:    fakeAPIFetcher([]byte(`[]`), nil),
			wantInText: []string{"No users found"},
		},
		"missing query": {
			params:     map[string]any{},
			fetcher:    fakeAPIFetcher(nil, nil),
			wantErr:    true,
			wantInText: []string{"query is required"},
		},
		"api error": {
			params:     map[string]any{"query": "alice"},
			fetcher:    fakeAPIFetcher(nil, errFake),
			wantErr:    true,
			wantInText: []string{"Failed to search users"},
		},
		"user without email": {
			params: map[string]any{"query": "bot"},
			fetcher: fakeAPIFetcher([]byte(`[
				{"accountId":"bot1","displayName":"Build Bot","emailAddress":""}
			]`), nil),
			wantInText: []string{"Build Bot", "accountId: bot1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			jiraAPIFetcher = tc.fetcher
			t.Cleanup(func() { jiraAPIFetcher = defaultJiraAPIFetch })

			result, err := handleSearchUsers(context.Background(), makeCallToolRequest(t, tc.params))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}

			text := resultText(t, result)
			if tc.wantErr && !result.IsError {
				t.Fatal("expected error result")
			}
			if !tc.wantErr && result.IsError {
				t.Fatalf("expected success, got error: %s", text)
			}
			for _, want := range tc.wantInText {
				if !strings.Contains(text, want) {
					t.Errorf("result missing %q\nfull: %s", want, text)
				}
			}
		})
	}
}

func TestHandleSearchUsersAPIPath(t *testing.T) {
	var capturedPath string
	jiraAPIFetcher = func(_ context.Context, _ string, path string) ([]byte, error) {
		capturedPath = path
		return []byte(`[]`), nil
	}
	t.Cleanup(func() { jiraAPIFetcher = defaultJiraAPIFetch })

	result, err := handleSearchUsers(context.Background(), makeCallToolRequest(t, map[string]any{
		"query":   "alice",
		"project": "PRO",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	if !strings.Contains(capturedPath, "project=PRO") {
		t.Errorf("path missing project param: %s", capturedPath)
	}
	if !strings.Contains(capturedPath, "query=alice") {
		t.Errorf("path missing query param: %s", capturedPath)
	}
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	var parts []string
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "\n")
}
