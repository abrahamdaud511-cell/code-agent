package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP servers",
	Long: `Manage MCP (Model Context Protocol) server connections.
MCP servers provide additional tools and capabilities to CodeAgent.`,
}

var mcpAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an MCP server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		command, _ := cmd.Flags().GetString("command")
		extraArgs := cmd.Flags().Args()[1:]

		if command == "" {
			return fmt.Errorf("--command is required")
		}

		cfg.AddMCPServer(name, command, extraArgs)
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Added MCP server: %s\n", name)
		return nil
	},
}

func init() {
	mcpCmd.AddCommand(mcpAddCmd)
	mcpAddCmd.Flags().StringP("command", "c", "", "Command to start the MCP server")
}
