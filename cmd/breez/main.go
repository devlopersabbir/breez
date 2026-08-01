package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/devlopersabbir/breez/internal/cli"
	"github.com/devlopersabbir/breez/internal/version"
	"github.com/spf13/cobra"
)

var (
	gatewayURL string
	subdomain  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "breez",
		Short: "Breez - Instant local tunnel CLI",
		Long:  "Expose your local development servers to the internet securely via Breez.",
	}

	rootCmd.PersistentFlags().StringVar(&gatewayURL, "gateway", "http://localhost:8080", "Gateway server URL")

	serveCmd := &cobra.Command{
		Use:   "serve <port>",
		Short: "Serve a local port over a public URL tunnel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[0])
			if err != nil || port <= 0 || port > 65535 {
				return fmt.Errorf("invalid port number: %s", args[0])
			}

			client := cli.NewClient(gatewayURL, port, subdomain)
			return client.Serve()
		},
	}

	serveCmd.Flags().StringVarP(&subdomain, "subdomain", "s", "", "Request a specific subdomain")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display CLI version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("breez v%s (commit: %s, built at: %s)\n", version.Version, version.Commit, version.BuildDate)
		},
	}

	rootCmd.AddCommand(serveCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
