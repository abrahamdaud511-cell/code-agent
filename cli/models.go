package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"codeagent/providers"
)

var modelsRefresh bool

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models from configured providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		providerRegistry := providers.NewRegistry()

		if modelsRefresh {
			if err := providerRegistry.Refresh(); err != nil {
				return fmt.Errorf("failed to refresh models: %w", err)
			}
		}

		models := providerRegistry.ListModels(cfg.Providers)
		if len(models) == 0 {
			fmt.Println("No models available. Configure a provider with 'codeagent auth login' or /connct")
			return nil
		}

		fmt.Println("Available models:")
		for _, m := range models {
			fmt.Printf("  - %s/%s\n", m.Provider, m.Name)
		}
		return nil
	},
}

func init() {
	modelsCmd.Flags().BoolVar(&modelsRefresh, "refresh", false, "Refresh cached model list")
}
