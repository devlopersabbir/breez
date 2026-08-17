package ui

import (
	"fmt"
	"time"

	"github.com/fatih/color"
)

// LogHTTP prints a formatted, color-coded HTTP log line.
func LogHTTP(method, path string, statusCode int, duration time.Duration) {
	now := time.Now().Format("15:04:05")
	timeStr := color.HiBlackString(now)

	// Method Badge
	var methodStr string
	switch method {
	case "GET":
		methodStr = color.New(color.FgGreen, color.Bold).Sprintf("%-6s", method)
	case "POST":
		methodStr = color.New(color.FgCyan, color.Bold).Sprintf("%-6s", method)
	case "PUT":
		methodStr = color.New(color.FgYellow, color.Bold).Sprintf("%-6s", method)
	case "DELETE":
		methodStr = color.New(color.FgRed, color.Bold).Sprintf("%-6s", method)
	case "PATCH":
		methodStr = color.New(color.FgMagenta, color.Bold).Sprintf("%-6s", method)
	default:
		methodStr = color.New(color.FgHiWhite, color.Bold).Sprintf("%-6s", method)
	}

	// Status Code
	var statusStr string
	switch {
	case statusCode >= 200 && statusCode < 300:
		statusStr = color.GreenString("[%d OK]", statusCode)
	case statusCode >= 300 && statusCode < 400:
		statusStr = color.CyanString("[%d Redirect]", statusCode)
	case statusCode >= 400 && statusCode < 500:
		statusStr = color.YellowString("[%d Client Error]", statusCode)
	default:
		statusStr = color.RedString("[%d Server Error]", statusCode)
	}

	// Latency
	var latStr string
	ms := float64(duration.Microseconds()) / 1000.0
	switch {
	case ms < 50.0:
		latStr = color.GreenString("%6.1fms", ms)
	case ms < 200.0:
		latStr = color.YellowString("%6.1fms", ms)
	default:
		latStr = color.RedString("%6.1fms", ms)
	}

	fmt.Printf("  %s  %s  %-32s %s  %s\n", timeStr, methodStr, color.HiWhiteString(path), statusStr, latStr)
}
