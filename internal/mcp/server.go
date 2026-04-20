package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates an MCP server with all Hatch tools registered.
func NewServer(version string) *server.MCPServer {
	s := server.NewMCPServer(
		"hatch",
		version,
	)

	client := NewDaemonClient()
	registerTools(s, client)

	return s
}

func registerTools(s *server.MCPServer, client *DaemonClient) {
	s.AddTool(
		mcp.NewTool("get_daemon_status",
			mcp.WithDescription("Check if the Hatch daemon is running. If it is not running, tell the user to run 'hatch up' in their terminal."),
		),
		handleGetDaemonStatus(client),
	)

	s.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithDescription("List all configured Hatch projects with their domains, services, health status, and process state. Use this to discover what projects are available."),
		),
		handleListProjects(client),
	)

	s.AddTool(
		mcp.NewTool("create_session",
			mcp.WithDescription("Create a new ephemeral session for a project. Allocates dynamic ports, starts the project's services, and returns HTTPS URLs where the session is accessible. Use this when you need to preview changes in a browser."),
			mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
			mcp.WithString("name", mcp.Description("Session name (becomes the subdomain, e.g. 'fix-auth' creates fix-auth.myapp.test). Auto-generated if omitted.")),
			mcp.WithNumber("ttl", mcp.Description("Time-to-live in seconds. Session is auto-destroyed after this much idle time. Default: 1800 (30 minutes).")),
		),
		handleCreateSession(client),
	)

	s.AddTool(
		mcp.NewTool("stop_session",
			mcp.WithDescription("Destroy a session, stop its processes, and release its ports."),
			mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
			mcp.WithString("name", mcp.Description("Session name"), mcp.Required()),
		),
		handleStopSession(client),
	)

	s.AddTool(
		mcp.NewTool("list_sessions",
			mcp.WithDescription("List all active sessions with their URLs, ports, and TTL status."),
		),
		handleListSessions(client),
	)

	s.AddTool(
		mcp.NewTool("restart_service",
			mcp.WithDescription("Restart a specific service within a project or session. Use this after making changes that require a service restart."),
			mcp.WithString("project", mcp.Description("Project name (for session services, use the qualified name: project~session)"), mcp.Required()),
			mcp.WithString("service", mcp.Description("Service name within the project"), mcp.Required()),
		),
		handleRestartService(client),
	)
}

func handleGetDaemonStatus(client *DaemonClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetStatus()
		if err != nil {
			return mcp.NewToolResultText("Hatch daemon is not running. Run 'hatch up' in your terminal to start it."), nil
		}
		var status map[string]any
		_ = json.Unmarshal(data, &status)
		status["running"] = true
		status["message"] = "Hatch daemon is running"
		return resultJSON(status)
	}
}

func handleListProjects(client *DaemonClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projects, err := client.ListProjects()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		health, _ := client.GetHealth()
		processes, _ := client.GetProcesses()

		result := map[string]json.RawMessage{
			"projects":  projects,
			"health":    health,
			"processes": processes,
		}
		return resultJSON(result)
	}
}

func handleCreateSession(client *DaemonClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		project := request.GetString("project", "")
		name := request.GetString("name", "")
		ttl := request.GetInt("ttl", 0)

		data, err := client.CreateSession(project, name, ttl)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func handleStopSession(client *DaemonClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		project := request.GetString("project", "")
		name := request.GetString("name", "")

		data, err := client.DestroySession(project, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func handleListSessions(client *DaemonClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.ListSessions()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func handleRestartService(client *DaemonClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		project := request.GetString("project", "")
		service := request.GetString("service", "")

		data, err := client.RestartProcess(project, service)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func resultJSON(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling result: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}
