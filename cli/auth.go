package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"codeagent/providers"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage provider authentication",
	Long: `Manage authentication credentials for LLM providers.
Supported providers: openai, anthropic, google, groq, openrouter, ollama, github-copilot

Usage:
  codeagent auth login --provider openai --key sk-...
  codeagent auth list
  codeagent auth remove openai`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with a provider",
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		key, _ := cmd.Flags().GetString("key")

		if provider == "" {
			return fmt.Errorf("provider is required")
		}
		if key == "" && provider != "ollama" {
			return fmt.Errorf("API key is required for %s", provider)
		}

		authFile := filepath.Join(os.Getenv("HOME"), ".local", "share", "codeagent", "auth.json")
		if _, err := os.Stat(authFile); os.IsNotExist(err) {
			os.MkdirAll(filepath.Dir(authFile), 0700)
		}

		creds := providers.LoadCredentials(authFile)
		creds[provider] = key
		if err := providers.SaveCredentials(authFile, creds); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		fmt.Printf("✓ Authenticated with %s\n", provider)
		return nil
	},
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		authFile := filepath.Join(os.Getenv("HOME"), ".local", "share", "codeagent", "auth.json")
		creds := providers.LoadCredentials(authFile)

		if len(creds) == 0 {
			fmt.Println("No providers configured. Use 'codeagent auth login' to add one.")
			return nil
		}

		fmt.Println("Configured providers:")
		for provider := range creds {
			fmt.Printf("  - %s\n", provider)
		}
		return nil
	},
}

var authRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a provider's credentials",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := args[0]
		authFile := filepath.Join(os.Getenv("HOME"), ".local", "share", "codeagent", "auth.json")
		creds := providers.LoadCredentials(authFile)

		if _, ok := creds[provider]; !ok {
			return fmt.Errorf("provider %s not configured", provider)
		}

		delete(creds, provider)
		if err := providers.SaveCredentials(authFile, creds); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		fmt.Printf("✓ Removed %s credentials\n", provider)
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authRemoveCmd)

	authLoginCmd.Flags().StringP("provider", "p", "", "Provider name")
	authLoginCmd.Flags().StringP("key", "k", "", "API key")
}
