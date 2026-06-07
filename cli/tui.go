package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"codeagent/core/session"
	"codeagent/core/tui"
)

var (
	continueSession bool
	sessionID      string
	forkSession    bool
	promptText     string
	modelFlag      string
	agentFlag      string
	portFlag       int
	hostnameFlag   string
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start the CodeAgent terminal user interface",
	Long: `Start the CodeAgent terminal user interface for an interactive
AI-powered coding session in your terminal.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionStore, err := session.NewStore(cfg.DataDir)
		if err != nil {
			return fmt.Errorf("failed to create session store: %w", err)
		}
		defer sessionStore.Close()

		var sess *session.Session
		if continueSession || sessionID != "" {
			sess, err = sessionStore.Load(sessionID)
			if err != nil {
				return fmt.Errorf("failed to load session: %w", err)
			}
			if forkSession {
				sess = sess.Fork()
			}
		} else {
			sess = session.New(cfg, sessionStore)
		}

		app, err := tui.New(cfg, sess, sessionStore)
		if err != nil {
			return fmt.Errorf("failed to create TUI: %w", err)
		}

		if promptText != "" {
			app.SetInitialPrompt(promptText)
		}

		if _, err := app.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	tuiCmd.Flags().BoolVarP(&continueSession, "continue", "c", false, "Continue the last session")
	tuiCmd.Flags().StringVarP(&sessionID, "session", "s", "", "Session ID to continue")
	tuiCmd.Flags().BoolVar(&forkSession, "fork", false, "Fork the session when continuing")
	tuiCmd.Flags().StringVar(&promptText, "prompt", "", "Prompt to use")
	tuiCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Model to use (provider/model)")
	tuiCmd.Flags().StringVar(&agentFlag, "agent", "", "Agent to use")
	tuiCmd.Flags().IntVar(&portFlag, "port", 0, "Port to listen on")
	tuiCmd.Flags().StringVar(&hostnameFlag, "hostname", "", "Hostname to listen on")
}
