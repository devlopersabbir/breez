package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devlopersabbir/breez/internal/protocol"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

type Client struct {
	GatewayURL string
	LocalPort  int
	Subdomain  string
}

func NewClient(gatewayURL string, localPort int, subdomain string) *Client {
	return &Client{
		GatewayURL: gatewayURL,
		LocalPort:  localPort,
		Subdomain:  subdomain,
	}
}

func (c *Client) Serve() error {
	u, err := url.Parse(c.GatewayURL)
	if err != nil {
		return fmt.Errorf("invalid gateway URL: %w", err)
	}

	wsScheme := "ws"
	if u.Scheme == "https" {
		wsScheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/_breez/ws", wsScheme, u.Host)

	boldCyan := color.New(color.FgCyan, color.Bold).SprintfFunc()
	boldGreen := color.New(color.FgGreen, color.Bold).SprintfFunc()
	boldYellow := color.New(color.FgYellow, color.Bold).SprintfFunc()
	dim := color.New(color.FgHiBlack).SprintfFunc()

	fmt.Println(boldCyan("\n  ☁  Breez Local Tunnel"))
	fmt.Println(dim("  ---------------------------------------------"))
	fmt.Printf("  %s Connecting to %s...\n", color.YellowString("➜"), wsURL)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to gateway: %w", err)
	}
	defer conn.Close()

	// Send init payload
	initPayload, _ := json.Marshal(protocol.TunnelInitPayload{
		RequestedSubdomain: c.Subdomain,
		LocalPort:          c.LocalPort,
	})

	initFrame, _ := json.Marshal(protocol.Frame{
		Type:    protocol.MsgTunnelInit,
		Payload: initPayload,
	})

	if err := conn.WriteMessage(websocket.TextMessage, initFrame); err != nil {
		return fmt.Errorf("failed to send init frame: %w", err)
	}

	// Read ready payload
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read ready frame: %w", err)
	}

	var frame protocol.Frame
	if err := json.Unmarshal(msgBytes, &frame); err != nil || frame.Type != protocol.MsgTunnelReady {
		return fmt.Errorf("unexpected frame received from gateway: %s", string(msgBytes))
	}

	var ready protocol.TunnelReadyPayload
	_ = json.Unmarshal(frame.Payload, &ready)

	fmt.Println()
	fmt.Println(boldGreen("  ✔ Tunnel Established Successfully!"))
	fmt.Println(dim("  ---------------------------------------------"))
	fmt.Printf("  %-12s %s\n", color.HiWhiteString("Local:"), color.CyanString("http://localhost:%d", c.LocalPort))
	fmt.Printf("  %-12s %s\n", color.HiWhiteString("Public:"), color.GreenString(ready.URL))
	fmt.Printf("  %-12s %s\n", color.HiWhiteString("Status:"), boldGreen("Online"))
	fmt.Println(dim("  ---------------------------------------------"))
	fmt.Println(boldYellow("  Requests Log:") + dim(" (Press Ctrl+C to stop)\n"))

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var reqFrame protocol.Frame
			if err := json.Unmarshal(message, &reqFrame); err != nil || reqFrame.Type != protocol.MsgRequest {
				continue
			}

			var req protocol.RequestPayload
			if err := json.Unmarshal(reqFrame.Payload, &req); err != nil {
				continue
			}

			go c.handleLocalRequest(conn, &req)
		}
	}()

	select {
	case <-interrupt:
		fmt.Println(color.YellowString("\n  Disconnecting tunnel..."))
	case <-done:
	}

	return nil
}

func (c *Client) handleLocalRequest(conn *websocket.Conn, req *protocol.RequestPayload) {
	startTime := time.Now()
	targetURL := fmt.Sprintf("http://localhost:%d%s", c.LocalPort, req.Path)

	httpReq, err := http.NewRequest(req.Method, targetURL, bytes.NewReader(req.Body))
	if err != nil {
		c.sendErrorResponse(conn, req.ID, http.StatusInternalServerError, err.Error())
		c.logRequest(req.Method, req.Path, http.StatusInternalServerError, time.Since(startTime))
		return
	}

	httpReq.Header = req.Headers

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.sendErrorResponse(conn, req.ID, http.StatusBadGateway, fmt.Sprintf("Local service error: %v", err))
		c.logRequest(req.Method, req.Path, http.StatusBadGateway, time.Since(startTime))
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	respPayload, _ := json.Marshal(protocol.ResponsePayload{
		ID:         req.ID,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	})

	respFrame, _ := json.Marshal(protocol.Frame{
		Type:    protocol.MsgResponse,
		Payload: respPayload,
	})

	_ = conn.WriteMessage(websocket.TextMessage, respFrame)
	c.logRequest(req.Method, req.Path, resp.StatusCode, time.Since(startTime))
}

func (c *Client) logRequest(method string, path string, statusCode int, duration time.Duration) {
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

	methodStr := color.New(color.FgHiWhite, color.Bold).Sprintf("%-6s", method)
	durationStr := color.HiBlackString("(%s)", duration.Round(time.Millisecond))

	fmt.Printf("  %s %s %s %s\n", methodStr, path, statusStr, durationStr)
}

func (c *Client) sendErrorResponse(conn *websocket.Conn, reqID string, status int, msg string) {
	respPayload, _ := json.Marshal(protocol.ResponsePayload{
		ID:         reqID,
		StatusCode: status,
		Headers:    make(http.Header),
		Body:       []byte(msg),
		Error:      msg,
	})

	respFrame, _ := json.Marshal(protocol.Frame{
		Type:    protocol.MsgResponse,
		Payload: respPayload,
	})

	_ = conn.WriteMessage(websocket.TextMessage, respFrame)
}
