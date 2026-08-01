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
		Short: "Breez - Instant, lightweight local tunnel CLI",
		Long: `☁  Breez CLI
  Expose your local development HTTP servers to the public internet instantly.

  Examples:
    $ breez serve 3000
    $ breez serve 8080 --subdomain myapp
    $ breez serve 5173 --gateway https://gateway.breez.run`,
	}

	rootCmd.PersistentFlags().StringVarP(&gatewayURL, "gateway", "g", "http://localhost:8080", "Gateway server URL to connect with")

	serveCmd := &cobra.Command{
		Use:   "serve <port>",
		Short: "Create a public tunnel forwarding to your local HTTP server port",
		Long: `Creates a secure WebSocket tunnel between your local port and the Breez Gateway server.

Arguments:
  <port>    The local port number where your application/server is running (e.g. 3000, 8080, 5173).

Examples:
  $ breez serve 3000
  $ breez serve 8080 --subdomain custom-name
  $ breez serve 5000 --gateway http://localhost:8080`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[0])
			if err != nil || port <= 0 || port > 65535 {
				return fmt.Errorf("invalid port number: %s. Port must be an integer between 1 and 65535", args[0])
			}

			client := cli.NewClient(gatewayURL, port, subdomain)
			return client.Serve()
		},
	}

	serveCmd.Flags().StringVarP(&subdomain, "subdomain", "s", "", "Request a custom subdomain name (e.g. --subdomain myapp)")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display version, commit hash, and build timestamp",
		Long:  "Displays the current installed Breez CLI version, git commit hash, and build timestamp.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("breez v%s (commit: %s, built at: %s)\n", version.Version, version.Commit, version.BuildDate)
		},
	}

	rootCmd.AddCommand(serveCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
