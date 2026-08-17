package ui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

var (
	cyanBold   = color.New(color.FgCyan, color.Bold).SprintFunc()
	greenBold  = color.New(color.FgGreen, color.Bold).SprintFunc()
	yellowBold = color.New(color.FgYellow, color.Bold).SprintFunc()
	whiteBold  = color.New(color.FgHiWhite, color.Bold).SprintFunc()
	dim        = color.New(color.FgHiBlack).SprintFunc()
)

// PrintLocalBanner prints a styled UI box for local mode (breez start).
func PrintLocalBanner(hostname, targetURL, version string, dnsPort int) {
	width := 64

	fmt.Println()
	fmt.Println(dim("  ┌" + strings.Repeat("─", width-4) + "┐"))
	fmt.Printf("  │  %s  %s%s%s  │\n",
		cyanBold("☁  BREEZ"),
		dim("v"+version),
		strings.Repeat(" ", width-35-len(version)),
		greenBold("● Online (Local)"),
	)
	fmt.Println(dim("  ├" + strings.Repeat("─", width-4) + "┤"))
	fmt.Printf("  │  %-14s %s%s│\n", dim("Local Domain:"), cyanBold(hostname), strings.Repeat(" ", max(0, width-20-len(hostname))))
	fmt.Printf("  │  %-14s %s%s│\n", dim("Target Port:"), whiteBold(targetURL), strings.Repeat(" ", max(0, width-20-len(targetURL))))
	dnsInfo := fmt.Sprintf("127.0.0.1:%d (*.breez.local)", dnsPort)
	fmt.Printf("  │  %-14s %s%s│\n", dim("DNS Resolver:"), yellowBold(dnsInfo), strings.Repeat(" ", max(0, width-20-len(dnsInfo))))
	fmt.Println(dim("  ├" + strings.Repeat("─", width-4) + "┤"))
	shortcuts := "[o] Open in Browser   [c] Copy URL   [q] Quit"
	fmt.Printf("  │  %s%s│\n", dim(shortcuts), strings.Repeat(" ", max(0, width-8-len(shortcuts))))
	fmt.Println(dim("  └" + strings.Repeat("─", width-4) + "┘"))
	fmt.Println()
	fmt.Println(yellowBold("  Live Request Logs:") + dim(" (monitoring local traffic...)\n"))
}

// PrintDualBanner prints a styled UI box for dual mode (breez serve with local DNS & public tunnel).
func PrintDualBanner(localURL, publicURL, targetURL, version string) {
	width := 68

	fmt.Println()
	fmt.Println(dim("  ┌" + strings.Repeat("─", width-4) + "┐"))
	fmt.Printf("  │  %s  %s%s%s  │\n",
		cyanBold("☁  BREEZ DUAL TUNNEL"),
		dim("v"+version),
		strings.Repeat(" ", width-46-len(version)),
		greenBold("● Connected"),
	)
	fmt.Println(dim("  ├" + strings.Repeat("─", width-4) + "┤"))
	if localURL != "" {
		fmt.Printf("  │  %-14s %s%s│\n", dim("Local URL:"), cyanBold(localURL), strings.Repeat(" ", max(0, width-20-len(localURL))))
	}
	fmt.Printf("  │  %-14s %s%s│\n", dim("Public URL:"), greenBold(publicURL), strings.Repeat(" ", max(0, width-20-len(publicURL))))
	fmt.Printf("  │  %-14s %s%s│\n", dim("Target:"), whiteBold(targetURL), strings.Repeat(" ", max(0, width-20-len(targetURL))))
	fmt.Println(dim("  ├" + strings.Repeat("─", width-4) + "┤"))
	shortcuts := "[o] Open Public   [l] Open Local   [c] Copy URL   [q] Stop"
	fmt.Printf("  │  %s%s│\n", dim(shortcuts), strings.Repeat(" ", max(0, width-8-len(shortcuts))))
	fmt.Println(dim("  └" + strings.Repeat("─", width-4) + "┘"))
	fmt.Println()
	fmt.Println(yellowBold("  Live Request Logs:") + dim(" (Ctrl+C to stop)\n"))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
