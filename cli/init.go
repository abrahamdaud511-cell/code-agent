package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initProjectCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize CodeAgent for a project",
	Long: `Initialize CodeAgent for the current project.
This analyzes your project and creates an AGENTS.md file
with project context for the AI agent.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()
		if len(args) > 0 {
			dir = args[0]
		}

		agentFile := filepath.Join(dir, "AGENTS.md")
		if _, err := os.Stat(agentFile); err == nil {
			return fmt.Errorf("AGENTS.md already exists in %s", dir)
		}

		projectName := filepath.Base(dir)

		content := fmt.Sprintf(`# %s

## Project Overview

This is the %s project.

## Tech Stack

- Language: 
- Framework: 
- Build System: 

## Conventions

- 

## Architecture

- 

## Commands

- Build: 
- Test: 
- Lint: 
- Typecheck: 
`, projectName, projectName)

		if err := os.WriteFile(agentFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create AGENTS.md: %w", err)
		}

		fmt.Printf("✓ Created AGENTS.md in %s\n", dir)
		return nil
	},
}
