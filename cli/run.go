package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"codeagent/core/agent"
	"codeagent/providers"
	"codeagent/core/session"
)

var (
	runCommand   string
	runContinue  bool
	runSession   string
	runFork      bool
	runShare     bool
	runModel     string
	runAgent     string
	runFile      []string
	runFormat    string
	runTitle     string
	runAttach    string
	runDir       string
	runPort      int
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run CodeAgent in non-interactive mode",
	Long: `Run CodeAgent in non-interactive mode by passing a prompt directly.
Useful for scripting and automation.

Examples:
  codeagent run "Explain this codebase"
  codeagent run --model openai/gpt-5 "Add error handling"
  codeagent run -f main.go "Review this file"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && runCommand == "" {
			return fmt.Errorf("prompt or --command is required")
		}

		prompt := runCommand
		if len(args) > 0 {
			prompt = args[0]
		}

		sessionStore, err := session.NewStore(cfg.DataDir)
		if err != nil {
			return fmt.Errorf("failed to create session store: %w", err)
		}
		defer sessionStore.Close()

		var sess *session.Session
		if runContinue || runSession != "" {
			sess, err = sessionStore.Load(runSession)
			if err != nil {
				return fmt.Errorf("failed to load session: %w", err)
			}
			if runFork {
				sess = sess.Fork()
			}
		} else {
			sess = session.New(cfg, sessionStore)
		}

		if runTitle != "" {
			sess.Title = runTitle
		}

		for _, f := range runFile {
			sess.AttachFile(f)
		}

		provider, err := providers.GetProvider(cfg, sess.Model)
		if err != nil {
			return fmt.Errorf("failed to create provider: %w", err)
		}

		ag, err := agent.New(cfg, sess, provider)
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		response, err := ag.Run(prompt)
		if err != nil {
			return fmt.Errorf("agent run failed: %w", err)
		}

		if runFormat == "json" {
			fmt.Println(response.JSON())
		} else {
			fmt.Println(response.Text)
		}

		if runShare {
			shareURL, err := sess.Share()
			if err == nil {
				fmt.Fprintf(os.Stderr, "Shared: %s\n", shareURL)
			}
		}

		return nil
	},
}

func init() {
	runCmd.Flags().StringVar(&runCommand, "command", "", "The command to run")
	runCmd.Flags().BoolVarP(&runContinue, "continue", "c", false, "Continue the last session")
	runCmd.Flags().StringVarP(&runSession, "session", "s", "", "Session ID to continue")
	runCmd.Flags().BoolVar(&runFork, "fork", false, "Fork the session when continuing")
	runCmd.Flags().BoolVar(&runShare, "share", false, "Share the session")
	runCmd.Flags().StringVarP(&runModel, "model", "m", "", "Model to use (provider/model)")
	runCmd.Flags().StringVar(&runAgent, "agent", "", "Agent to use")
	runCmd.Flags().StringSliceVarP(&runFile, "file", "f", nil, "Files to attach")
	runCmd.Flags().StringVar(&runFormat, "format", "default", "Output format (default or json)")
	runCmd.Flags().StringVar(&runTitle, "title", "", "Session title")
}
