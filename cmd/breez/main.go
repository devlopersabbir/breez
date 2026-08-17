package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/devlopersabbir/breez/internal/cli"
	"github.com/devlopersabbir/breez/internal/dns"
	"github.com/devlopersabbir/breez/internal/registry"
	"github.com/devlopersabbir/breez/internal/router"
	"github.com/devlopersabbir/breez/internal/ui"
	"github.com/devlopersabbir/breez/internal/version"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	gatewayURL string
	subdomain  string
	localName  string
	dnsPort    int
	httpPort   int
	domain     string
	enableDNS  bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "breez",
		Short: "Breez - Local DNS & Public Tunnel Platform for Developers",
		Long: `☁  BREEZ PLATFORM
  Instant local domain resolution (*.breez.local) & public tunneling for developers.

  Quick Start:
    $ breez start 3000                  # Local domain (http://<name>.breez.local)
    $ breez start 3000 --name myapp     # Named local domain (http://myapp.breez.local)
    $ breez serve 3000                  # Public internet tunnel
    $ breez list                        # List active routes
    $ breez dns setup                   # One-time macOS resolver setup`,
	}

	rootCmd.PersistentFlags().StringVarP(&gatewayURL, "gateway", "g", "http://localhost:8080", "Gateway server URL to connect with")
	rootCmd.PersistentFlags().StringVar(&domain, "domain", "breez.local", "Local base domain for DNS resolution")

	// 1. `breez start <port>` -> Local development mode (DNS + Reverse Proxy)
	startCmd := &cobra.Command{
		Use:   "start <port>",
		Short: "Assign a clean local domain (*.breez.local) to your local port",
		Long: `Starts a local session mapping http://<name>.breez.local to your local development server port.
Zero latency, runs locally without needing public internet connection.

Arguments:
  <port>    The local port where your application is running (e.g. 3000, 8080, 5173).

Examples:
  $ breez start 3000
  $ breez start 8080 --name myapi
  $ breez start 3000 --dns-port 5354 --http-port 8080`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[0])
			if err != nil || port <= 0 || port > 65535 {
				return fmt.Errorf("invalid port number: %s (must be between 1 and 65535)", args[0])
			}

			return cli.RunLocal(cli.LocalOptions{
				Port:     port,
				Name:     localName,
				DNSPort:  dnsPort,
				HTTPPort: httpPort,
				Domain:   domain,
			})
		},
	}
	startCmd.Flags().StringVarP(&localName, "name", "n", "", "Custom local subdomain name (e.g. --name myapp -> myapp.breez.local)")
	startCmd.Flags().IntVar(&dnsPort, "dns-port", 53, "Local DNS server port (default: 53)")
	startCmd.Flags().IntVar(&httpPort, "http-port", 80, "Local HTTP Router port (default: 80)")

	// 2. `breez serve <port>` -> Public Tunnel (WebSocket Gateway)
	serveCmd := &cobra.Command{
		Use:   "serve <port>",
		Short: "Create a public tunnel forwarding to your local HTTP server port",
		Long: `Creates a secure WebSocket tunnel between your local port and the Breez Gateway server.
Also automatically registers a local *.breez.local domain if local DNS daemon is available.

Arguments:
  <port>    The local port where your server is running (e.g. 3000, 8080, 5173).

Examples:
  $ breez serve 3000
  $ breez serve 8080 --subdomain custom-name
  $ breez serve 5000 --gateway https://gateway.breez.run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[0])
			if err != nil || port <= 0 || port > 65535 {
				return fmt.Errorf("invalid port number: %s. Port must be an integer between 1 and 65535", args[0])
			}

			client := cli.NewClient(gatewayURL, port, subdomain)
			client.EnableDNS = enableDNS
			client.Domain = domain
			return client.Serve()
		},
	}
	serveCmd.Flags().StringVarP(&subdomain, "subdomain", "s", "", "Request a custom subdomain name (e.g. --subdomain myapp)")
	serveCmd.Flags().BoolVar(&enableDNS, "dns", true, "Register local breez.local domain alongside public tunnel")

	// 3. `breez list` -> List active routes
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all active local routes and tunnels",
		Long:  "Fetches and displays a table of all actively registered local domains and routes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ipcClient := registry.NewIPCClient()
			if !ipcClient.Ping() {
				fmt.Println(color.YellowString("\n  No active Breez daemon detected."))
				fmt.Println(color.HiBlackString("  Start a route with: ") + color.CyanString("breez start 3000 --name myapp\n"))
				return nil
			}

			routes, err := ipcClient.ListRoutes()
			if err != nil {
				return fmt.Errorf("failed to fetch active routes: %w", err)
			}
			ui.PrintRouteTable(routes)
			return nil
		},
	}

	// 4. `breez stop <name>` -> Stop active route
	stopCmd := &cobra.Command{
		Use:   "stop <name>",
		Short: "Deregister a running local route by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ipcClient := registry.NewIPCClient()
			if !ipcClient.Ping() {
				return fmt.Errorf("no active Breez daemon found")
			}
			if err := ipcClient.DeregisterRoute(args[0]); err != nil {
				return fmt.Errorf("failed to stop route '%s': %w", args[0], err)
			}
			fmt.Printf("  %s Route '%s' deregistered successfully.\n", color.GreenString("✔"), args[0])
			return nil
		},
	}

	// 5. `breez dns` -> Subcommands for DNS resolver management
	dnsCmd := &cobra.Command{
		Use:   "dns",
		Short: "Manage local DNS resolver (*.breez.local)",
		Long:  "Commands to start, inspect, or configure local system DNS resolution for Breez.",
	}

	dnsStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Run the local DNS server and HTTP router daemon in foreground",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			reg := registry.New()
			ipcServer, err := registry.StartIPCServer(reg)
			if err != nil {
				return fmt.Errorf("failed to start IPC control server: %w", err)
			}
			defer ipcServer.Stop()

			// Start DNS
			dnsAddr := fmt.Sprintf("127.0.0.1:%d", dnsPort)
			dnsSrv := dns.NewServer(dns.Config{
				Domain:   domain,
				BindAddr: dnsAddr,
				TargetIP: net.ParseIP("127.0.0.1"),
			})
			go func() {
				if err := dnsSrv.Start(ctx); err != nil {
					fmt.Printf("%s DNS Server error on %s: %v\n", color.RedString("✘"), dnsAddr, err)
				}
			}()
			defer dnsSrv.Stop()

			// Start HTTP Router
			httpAddr := fmt.Sprintf("127.0.0.1:%d", httpPort)
			httpRouter := router.New(router.Config{
				BindAddr: httpAddr,
				Domain:   domain,
			}, reg)
			go func() {
				if err := httpRouter.Start(ctx); err != nil {
					fmt.Printf("%s HTTP Router error on %s: %v\n", color.RedString("✘"), httpAddr, err)
				}
			}()
			defer httpRouter.Stop()

			fmt.Println(color.New(color.FgCyan, color.Bold).Sprint("\n  ☁  BREEZ LOCAL DAEMON RUNNING"))
			fmt.Println(color.HiBlackString("  ---------------------------------------------"))
			fmt.Printf("  %-16s %s\n", color.HiBlackString("DNS Server:"), color.GreenString("127.0.0.1:%d (%s)", dnsPort, domain))
			fmt.Printf("  %-16s %s\n", color.HiBlackString("HTTP Router:"), color.GreenString("127.0.0.1:%d", httpPort))
			fmt.Printf("  %-16s %s\n", color.HiBlackString("IPC Socket:"), color.HiWhiteString(registry.GetSocketPath()))
			fmt.Println(color.HiBlackString("  ---------------------------------------------"))
			fmt.Println(color.YellowString("  Press Ctrl+C to stop daemon.\n"))

			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
			<-interrupt
			fmt.Println(color.YellowString("\n  Shutting down Breez daemon..."))
			return nil
		},
	}
	dnsStartCmd.Flags().IntVar(&dnsPort, "port", 53, "DNS listen port (53 or 5354)")
	dnsStartCmd.Flags().IntVar(&httpPort, "http-port", 80, "HTTP router listen port (80 or 8080)")

	dnsSetupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure OS resolver to route *.breez.local queries locally (macOS)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("  %s Configuring system resolver for '*.%s'...\n", color.YellowString("➜"), domain)
			if err := dns.InstallMacResolver(domain, dnsPort); err != nil {
				return err
			}
			fmt.Printf("  %s Successfully configured /etc/resolver/%s pointing to 127.0.0.1:%d\n",
				color.GreenString("✔"), domain, dnsPort)
			fmt.Printf("  %s Test resolution with: %s\n\n",
				color.CyanString("➜"), color.HiWhiteString("ping test.%s", domain))
			return nil
		},
	}
	dnsSetupCmd.Flags().IntVar(&dnsPort, "port", 53, "DNS port configured in resolver file")

	dnsStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Diagnose local DNS resolution status",
		Run: func(cmd *cobra.Command, args []string) {
			dnsAddr := fmt.Sprintf("127.0.0.1:%d", dnsPort)
			st := dns.CheckStatus(domain, dnsAddr)

			fmt.Println(color.New(color.FgCyan, color.Bold).Sprint("\n  ☁  BREEZ DNS DIAGNOSTICS"))
			fmt.Println(color.HiBlackString("  ---------------------------------------------"))
			fmt.Printf("  %-20s %s\n", color.HiBlackString("Operating System:"), color.HiWhiteString(st.OS))
			fmt.Printf("  %-20s %s\n", color.HiBlackString("Domain Zone:"), color.HiWhiteString(st.Domain))
			if st.ResolverConfig != "" {
				cfgStatus := color.RedString("Missing")
				if st.ConfigExists {
					cfgStatus = color.GreenString("Configured (%s)", st.ResolverConfig)
				}
				fmt.Printf("  %-20s %s\n", color.HiBlackString("OS Resolver File:"), cfgStatus)
			}

			if st.ResolvingLocal {
				fmt.Printf("  %-20s %s\n", color.HiBlackString("Local Query Test:"), color.GreenString("OK -> %s", st.ResolvedIP))
			} else {
				fmt.Printf("  %-20s %s\n", color.HiBlackString("Local Query Test:"), color.YellowString("Failed (DNS daemon not running on %s)", dnsAddr))
			}
			fmt.Println(color.HiBlackString("  ---------------------------------------------\n"))
		},
	}
	dnsStatusCmd.Flags().IntVar(&dnsPort, "port", 53, "DNS port to query for diagnostics")

	dnsCmd.AddCommand(dnsStartCmd, dnsSetupCmd, dnsStatusCmd)

	// 6. Version Command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display version, commit hash, and build timestamp",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("breez v%s (commit: %s, built at: %s)\n", version.Version, version.Commit, version.BuildDate)
		},
	}

	rootCmd.AddCommand(startCmd, serveCmd, listCmd, stopCmd, dnsCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
