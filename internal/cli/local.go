package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/devlopersabbir/breez/internal/dns"
	"github.com/devlopersabbir/breez/internal/registry"
	"github.com/devlopersabbir/breez/internal/router"
	"github.com/devlopersabbir/breez/internal/ui"
	"github.com/devlopersabbir/breez/internal/version"
	"github.com/fatih/color"
)

// LocalOptions holds parameters for `breez start`.
type LocalOptions struct {
	Port      int
	Name      string
	DNSPort   int
	HTTPPort  int
	Domain    string
}

// RunLocal initiates a local DNS + reverse proxy session.
func RunLocal(opts LocalOptions) error {
	if opts.Domain == "" {
		opts.Domain = "breez.local"
	}
	if opts.DNSPort == 0 {
		opts.DNSPort = 53
	}
	if opts.HTTPPort == 0 {
		opts.HTTPPort = 80
	}
	if opts.Name == "" {
		opts.Name = generateRandomName()
	}
	opts.Name = strings.ToLower(opts.Name)

	fullHostname := fmt.Sprintf("%s.%s", opts.Name, opts.Domain)
	targetURL := fmt.Sprintf("http://localhost:%d", opts.Port)

	ipcClient := registry.NewIPCClient()
	isDaemonRunning := ipcClient.Ping()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var reg *registry.Registry
	var route *registry.Route

	if isDaemonRunning {
		// Register with running background daemon
		var err error
		route, err = ipcClient.RegisterRoute(opts.Name, opts.Domain, opts.Port)
		if err != nil {
			return fmt.Errorf("failed to register with local Breez daemon: %w", err)
		}
		defer func() {
			_ = ipcClient.DeregisterRoute(opts.Name)
		}()
	} else {
		// Start embedded DNS and Router within this process
		reg = registry.New()
		var err error
		route, err = reg.Register(opts.Name, opts.Domain, opts.Port)
		if err != nil {
			return err
		}

		// Try binding DNS port (e.g. 53, fallback to 5354 if permission denied)
		dnsBindAddr := fmt.Sprintf("127.0.0.1:%d", opts.DNSPort)
		dnsSrv := dns.NewServer(dns.Config{
			Domain:   opts.Domain,
			BindAddr: dnsBindAddr,
			TargetIP: net.ParseIP("127.0.0.1"),
		})

		go func() {
			if err := dnsSrv.Start(ctx); err != nil {
				// If port 53 fails due to permissions, warn the user
				if opts.DNSPort == 53 {
					fmt.Printf("%s Could not bind DNS on port 53 (requires sudo). Try: 'sudo breez start %d' or use --dns-port 5354\n",
						color.YellowString("⚠"), opts.Port)
				}
			}
		}()
		defer dnsSrv.Stop()

		// Start HTTP Reverse Proxy Router
		httpBindAddr := fmt.Sprintf("127.0.0.1:%d", opts.HTTPPort)
		httpRouter := router.New(router.Config{
			BindAddr: httpBindAddr,
			Domain:   opts.Domain,
		}, reg)

		go func() {
			if err := httpRouter.Start(ctx); err != nil {
				if opts.HTTPPort == 80 {
					fmt.Printf("%s Could not bind HTTP Router on port 80 (requires sudo). Try: 'sudo breez start %d' or use --http-port 8080\n",
						color.YellowString("⚠"), opts.Port)
				}
			}
		}()
		defer httpRouter.Stop()

		// Also start IPC server so `breez list` works
		ipcServer, err := registry.StartIPCServer(reg)
		if err == nil {
			defer ipcServer.Stop()
		}
	}

	displayLocalURL := fmt.Sprintf("http://%s", fullHostname)
	if opts.HTTPPort != 80 && !isDaemonRunning {
		displayLocalURL = fmt.Sprintf("http://%s:%d", fullHostname, opts.HTTPPort)
	}

	ui.PrintLocalBanner(displayLocalURL, targetURL, version.Version, opts.DNSPort)

	// Intercept local traffic logging via a lightweight monitoring listener if embedded
	startLogMonitor(ctx, route, opts.Port)

	// Wait for interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case <-interrupt:
		fmt.Println(color.YellowString("\n  Stopping local route '%s'...", fullHostname))
	case <-ctx.Done():
	}

	return nil
}

func startLogMonitor(ctx context.Context, route *registry.Route, targetPort int) {
	// A logging reverse proxy wrapper for direct hit observation
	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", targetPort))
	proxy := httputil.NewSingleHostReverseProxy(proxyURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		proxy.ServeHTTP(rec, r)
		ui.LogHTTP(r.Method, r.URL.RequestURI(), rec.statusCode, time.Since(start))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func generateRandomName() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
