package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"codeagent/config"
)

var (
	agentPath        string
	agentDescription string
	agentMode        string
	agentPermissions []string
	agentModel       string
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents for CodeAgent",
	Long: `Manage custom agents for CodeAgent.
Agents have custom system prompts and permission configurations.`,
}

var agentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		defaultPath := filepath.Join(home, ".config", "codeagent", "agents")

		if agentPath == "" {
			agentPath = defaultPath
		}

		os.MkdirAll(agentPath, 0755)

		if agentPermissions == nil {
			agentPermissions = []string{"bash", "read", "edit", "glob", "grep", "webfetch", "task", "todowrite"}
		}

		if agentModel == "" {
			agentModel = "openai/gpt-5"
		}

		ag := &config.AgentConfig{
			Name:        args[0],
			Description: agentDescription,
			Mode:        agentMode,
			Permissions: agentPermissions,
			Model:       agentModel,
		}

		agentFile := filepath.Join(agentPath, args[0]+".md")
		content := fmt.Sprintf(`---
name: %s
description: %s
model: %s
mode: %s
permissions: %v
---

# %s Agent

%s
`, ag.Name, ag.Description, ag.Model, ag.Mode, ag.Permissions, ag.Name, ag.Description)

		if err := os.WriteFile(agentFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		fmt.Printf("✓ Agent '%s' created at %s\n", ag.Name, agentFile)
		return nil
	},
}

func init() {
	agentCmd.AddCommand(agentCreateCmd)
	agentCreateCmd.Flags().StringVar(&agentPath, "path", "", "Directory to write the agent file")
	agentCreateCmd.Flags().StringVar(&agentDescription, "description", "", "What the agent should do")
	agentCreateCmd.Flags().StringVar(&agentMode, "mode", "primary", "Agent mode: all, primary, or subagent")
	agentCreateCmd.Flags().StringSliceVar(&agentPermissions, "permissions", nil, "Comma-separated permissions")
	agentCreateCmd.Flags().StringVarP(&agentModel, "model", "m", "", "Model to use (provider/model)")
}
