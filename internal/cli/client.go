package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devlopersabbir/breez/internal/protocol"
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

	fmt.Printf("Connecting to Breez Gateway at %s...\n", wsURL)

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

	fmt.Println("\n✔ Tunnel Created Successfully!")
	fmt.Printf("Local:  http://localhost:%d\n", c.LocalPort)
	fmt.Printf("Public: %s\n", ready.URL)
	fmt.Println("Status: Connected (Press Ctrl+C to stop)")
	fmt.Println(stringsRepeat("-", 45))

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("\n[CLI] Disconnected from gateway: %v", err)
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
		fmt.Println("\nStopping tunnel...")
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
		return
	}

	httpReq.Header = req.Headers

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.sendErrorResponse(conn, req.ID, http.StatusBadGateway, fmt.Sprintf("Local service error: %v", err))
		fmt.Printf("➜ %s %s [502 Bad Gateway] (%s)\n", req.Method, req.Path, time.Since(startTime).Round(time.Millisecond))
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
	fmt.Printf("➜ %s %s [%d] (%s)\n", req.Method, req.Path, resp.StatusCode, time.Since(startTime).Round(time.Millisecond))
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

func stringsRepeat(s string, count int) string {
	res := ""
	for i := 0; i < count; i++ {
		res += s
	}
	return res
}
