// gojira-mcp is an MCP (Model Context Protocol) server that exposes
// Jira operations as structured tools. It wraps jira-cli, providing
// AI assistants like Claude Code with a clean tool interface for
// creating, editing, searching, and managing Jira issues.
//
// Install jira-cli (https://github.com/ankitpokhrel/jira-cli) and
// configure it before running this server.
//
// Usage:
//
//	claude mcp add --scope user jira -- /path/to/gojira-mcp
package main

import (
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.SetOutput(os.Stderr)

	s := server.NewMCPServer(
		"gojira-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(createIssueTool, handleCreateIssue)
	s.AddTool(editIssueTool, handleEditIssue)
	s.AddTool(moveIssueTool, handleMoveIssue)
	s.AddTool(viewIssueTool, handleViewIssue)
	s.AddTool(listIssuesTool, handleListIssues)
	s.AddTool(addCommentTool, handleAddComment)
	s.AddTool(addToSprintTool, handleAddToSprint)
	s.AddTool(assignIssueTool, handleAssignIssue)
	s.AddTool(linkIssuesTool, handleLinkIssues)
	s.AddTool(unlinkIssuesTool, handleUnlinkIssues)
	s.AddTool(searchUsersTool, handleSearchUsers)
	s.AddTool(cloneIssueTool, handleCloneIssue)
	s.AddTool(deleteIssueTool, handleDeleteIssue)
	s.AddTool(watchIssueTool, handleWatchIssue)
	s.AddTool(addWorklogTool, handleAddWorklog)
	s.AddTool(createEpicTool, handleCreateEpic)
	s.AddTool(listEpicsTool, handleListEpics)
	s.AddTool(addIssuesToEpicTool, handleAddIssuesToEpic)
	s.AddTool(removeIssuesFromEpicTool, handleRemoveIssuesFromEpic)
	s.AddTool(listSprintsTool, handleListSprints)
	s.AddTool(listBoardsTool, handleListBoards)
	s.AddTool(listProjectsTool, handleListProjects)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
