package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/devlopersabbir/breez/internal/registry"
	"github.com/fatih/color"
)

// PrintRouteTable prints a clean formatted table of registered routes.
func PrintRouteTable(routes []*registry.Route) {
	if len(routes) == 0 {
		fmt.Println()
		fmt.Println(color.YellowString("  No active Breez routes currently running."))
		fmt.Println(color.HiBlackString("  Start a new local route with: ") + color.CyanString("breez start 3000 --name myapp"))
		fmt.Println()
		return
	}

	fmt.Println()
	fmt.Println(color.New(color.FgCyan, color.Bold).Sprint("  ☁  ACTIVE BREEZ ROUTES"))
	fmt.Println(color.HiBlackString("  " + strings.Repeat("─", 72)))
	fmt.Printf("  %-28s  %-14s  %-10s  %-14s\n",
		color.HiBlackString("LOCAL DOMAIN"),
		color.HiBlackString("TARGET PORT"),
		color.HiBlackString("REQUESTS"),
		color.HiBlackString("UPTIME"),
	)
	fmt.Println(color.HiBlackString("  " + strings.Repeat("─", 72)))

	now := time.Now()
	for _, r := range routes {
		uptime := now.Sub(r.CreatedAt).Round(time.Second)
		fmt.Printf("  %-28s  %-14s  %-10s  %-14s\n",
			color.CyanString("http://%s", r.Hostname),
			color.HiWhiteString("localhost:%d", r.TargetPort),
			color.GreenString("%d reqs", r.Requests),
			color.HiBlackString("%s", uptime),
		)
	}
	fmt.Println(color.HiBlackString("  " + strings.Repeat("─", 72)))
	fmt.Println()
}

// OpenBrowser opens the specified URL in the default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // linux, bsd
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
