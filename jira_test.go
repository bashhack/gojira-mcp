package main

import (
	"context"
	"testing"
)

func TestParseIssueKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard key in URL",
			input: "https://example.atlassian.net/browse/PROJ-456",
			want:  "PROJ-456",
		},
		{
			name:  "key in creation output",
			input: "✓ Issue created\nhttps://example.atlassian.net/browse/PROJ-1\n",
			want:  "PROJ-1",
		},
		{
			name:  "key with long project prefix",
			input: "Created MYPROJECT-12345",
			want:  "MYPROJECT-12345",
		},
		{
			name:  "no key present",
			input: "some random output with no key",
			want:  "",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "lowercase not matched",
			input: "proj-123 is not a key",
			want:  "",
		},
		{
			name:  "multiple keys returns first",
			input: "PROJ-1 and PROJ-2",
			want:  "PROJ-1",
		},
		{
			name:  "single letter project",
			input: "A-1",
			want:  "",
		},
		{
			name:  "two letter project",
			input: "AB-99",
			want:  "AB-99",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseIssueKey(tc.input)
			if got != tc.want {
				t.Errorf("parseIssueKey(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsIssueKey(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"PROJ-123", true},
		{"AB-1", true},
		{"MYPROJECT-99999", true},
		{"proj-123", false},
		{"A-1", false},
		{"", false},
		{"not a key", false},
		{"PROJ-123 extra", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if got := isIssueKey(tc.input); got != tc.want {
				t.Errorf("isIssueKey(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestFindActiveSprint(t *testing.T) {
	tests := []struct {
		name    string
		stdout  string
		stderr  string
		err     error
		wantID  string
		wantErr bool
	}{
		{
			name:   "single active sprint",
			stdout: "3601\n",
			wantID: "3601",
		},
		{
			name:   "sprint ID with whitespace",
			stdout: "  3601  \n",
			wantID: "3601",
		},
		{
			name:   "multiple sprints returns first",
			stdout: "100\n200\n",
			wantID: "100",
		},
		{
			name:    "empty output",
			stdout:  "",
			wantErr: true,
		},
		{
			name:    "only whitespace",
			stdout:  "   \n",
			wantErr: true,
		},
		{
			name:    "command error",
			stdout:  "",
			stderr:  "auth failed",
			err:     errFake,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jiraRunner = fakeRunner(tc.stdout, tc.stderr, tc.err)
			t.Cleanup(func() { jiraRunner = defaultRunJira })

			id, err := findActiveSprint(context.Background())
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tc.wantID {
				t.Errorf("got %q, want %q", id, tc.wantID)
			}
		})
	}
}
