package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	server "codeagent/api"
)

var (
	servePort    int
	serveHost    string
	serveMdns    bool
	serveMdnsDom string
	serveCors    []string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a headless CodeAgent server",
	Long: `Start a headless CodeAgent HTTP server for API access.
Set CODEAGENT_SERVER_PASSWORD to enable HTTP basic auth.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := server.New(cfg, server.Options{
			Port:         servePort,
			Hostname:     serveHost,
			EnableMdns:   serveMdns,
			MdnsDomain:   serveMdnsDom,
			CorsOrigins:  serveCors,
		})
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		fmt.Fprintf(os.Stderr, "CodeAgent server started on %s:%d\n", serveHost, servePort)
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
		return nil
	},
}

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start CodeAgent with a web interface",
	Long: `Start a headless CodeAgent server with a web interface.
Opens a browser to access CodeAgent through a web UI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := server.New(cfg, server.Options{
			Port:        servePort,
			Hostname:    serveHost,
			EnableWeb:   true,
			CorsOrigins: serveCors,
		})
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		fmt.Fprintf(os.Stderr, "CodeAgent web interface starting on http://%s:%d\n", serveHost, servePort)
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 4096, "Port to listen on")
	serveCmd.Flags().StringVar(&serveHost, "hostname", "localhost", "Hostname to listen on")
	serveCmd.Flags().BoolVar(&serveMdns, "mdns", false, "Enable mDNS discovery")
	serveCmd.Flags().StringVar(&serveMdnsDom, "mdns-domain", "", "Custom mDNS domain name")
	serveCmd.Flags().StringSliceVar(&serveCors, "cors", nil, "Additional CORS origins")

	webCmd.Flags().IntVar(&servePort, "port", 4096, "Port to listen on")
	webCmd.Flags().StringVar(&serveHost, "hostname", "localhost", "Hostname to listen on")
	webCmd.Flags().StringSliceVar(&serveCors, "cors", nil, "Additional CORS origins")
}
