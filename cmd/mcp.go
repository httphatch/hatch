package cmd

import (
	"os"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	hatchmcp "github.com/httphatch/hatch/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:    "mcp",
	Short:  "Start MCP server for AI agent integration",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := hatchmcp.NewServer(version)
		stdio := mcpserver.NewStdioServer(s)
		return stdio.Listen(cmd.Context(), os.Stdin, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
