package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"codeagent/config"
)

var (
	cfgFile   string
	logLevel  string
	printLogs bool
	pureMode  bool
	cfg       *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "codeagent",
	Short: "CodeAgent - The open source AI coding agent",
	Long: `CodeAgent is an open source AI coding agent that helps you write code
in your terminal. It supports 75+ LLM providers including OpenAI, Anthropic,
Google, Ollama, and more. Use /connct to add your API keys and start coding.

Complete documentation is available at https://codeagent.ai`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			runCmd.Run(cmd, args)
			return
		}
		tuiCmd.Run(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/codeagent/codeagent.json)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "INFO", "log level (DEBUG, INFO, WARN, ERROR)")
	rootCmd.PersistentFlags().BoolVar(&printLogs, "print-logs", false, "print logs to stderr")
	rootCmd.PersistentFlags().BoolVar(&pureMode, "pure", false, "run without external plugins")

	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(webCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(modelsCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(initProjectCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(filepath.Join(home, ".config", "codeagent"))
		viper.SetConfigType("json")
		viper.SetConfigName("codeagent")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		}
	}
}
