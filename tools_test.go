package main

import (
	"testing"
)

func TestToolSchemaRequiredFields(t *testing.T) {
	tests := map[string]struct {
		required []string
		want     []string
	}{
		"create_issue":  {required: createIssueTool.InputSchema.Required, want: []string{"summary", "type"}},
		"edit_issue":    {required: editIssueTool.InputSchema.Required, want: []string{"key"}},
		"move_issue":    {required: moveIssueTool.InputSchema.Required, want: []string{"key", "status"}},
		"view_issue":    {required: viewIssueTool.InputSchema.Required, want: []string{"key"}},
		"add_comment":   {required: addCommentTool.InputSchema.Required, want: []string{"key", "body"}},
		"add_to_sprint": {required: addToSprintTool.InputSchema.Required, want: []string{"key", "sprint"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if len(tc.required) != len(tc.want) {
				t.Fatalf("required fields count = %d, want %d", len(tc.required), len(tc.want))
			}
			for i, want := range tc.want {
				if tc.required[i] != want {
					t.Errorf("required[%d] = %q, want %q", i, tc.required[i], want)
				}
			}
		})
	}
}

func TestReadOnlyAnnotations(t *testing.T) {
	readOnly := map[string]*bool{
		"view_issue":  viewIssueTool.Annotations.ReadOnlyHint,
		"list_issues": listIssuesTool.Annotations.ReadOnlyHint,
	}
	for name, hint := range readOnly {
		t.Run(name, func(t *testing.T) {
			if hint == nil || !*hint {
				t.Errorf("%s should have ReadOnlyHint=true", name)
			}
		})
	}

	mutating := map[string]*bool{
		"create_issue":  createIssueTool.Annotations.ReadOnlyHint,
		"edit_issue":    editIssueTool.Annotations.ReadOnlyHint,
		"move_issue":    moveIssueTool.Annotations.ReadOnlyHint,
		"add_comment":   addCommentTool.Annotations.ReadOnlyHint,
		"add_to_sprint": addToSprintTool.Annotations.ReadOnlyHint,
	}
	for name, hint := range mutating {
		t.Run(name+"_not_readonly", func(t *testing.T) {
			if hint != nil && *hint {
				t.Errorf("%s should not have ReadOnlyHint=true", name)
			}
		})
	}
}

func TestAllToolsHaveDescriptions(t *testing.T) {
	tools := map[string]string{
		"create_issue":  createIssueTool.Description,
		"edit_issue":    editIssueTool.Description,
		"move_issue":    moveIssueTool.Description,
		"view_issue":    viewIssueTool.Description,
		"list_issues":   listIssuesTool.Description,
		"add_comment":   addCommentTool.Description,
		"add_to_sprint": addToSprintTool.Description,
	}

	for name, desc := range tools {
		t.Run(name, func(t *testing.T) {
			if desc == "" {
				t.Errorf("%s has no description", name)
			}
		})
	}
}
