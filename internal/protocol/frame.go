package protocol

import (
	"encoding/json"
	"net/http"
)

// MessageType identifies the frame type sent across WebSocket
type MessageType string

const (
	MsgTunnelInit  MessageType = "tunnel_init"  // CLI -> Gateway: Request new tunnel
	MsgTunnelReady MessageType = "tunnel_ready" // Gateway -> CLI: Tunnel established with public URL
	MsgRequest     MessageType = "request"      // Gateway -> CLI: Incoming HTTP request frame
	MsgResponse    MessageType = "response"     // CLI -> Gateway: HTTP response frame
	MsgPing        MessageType = "ping"         // Heartbeat ping
	MsgPong        MessageType = "pong"         // Heartbeat pong
	MsgError       MessageType = "error"        // Error frame
)

// Frame represents a generic message envelope exchanged over Gorilla WebSocket
type Frame struct {
	Type    MessageType     `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// TunnelInitPayload sent by CLI when requesting a tunnel
type TunnelInitPayload struct {
	RequestedSubdomain string `json:"requested_subdomain,omitempty"`
	LocalPort          int    `json:"local_port"`
	AuthToken          string `json:"auth_token,omitempty"`
}

// TunnelReadyPayload sent by Gateway when tunnel is assigned
type TunnelReadyPayload struct {
	Subdomain string `json:"subdomain"`
	URL       string `json:"url"`
	TunnelID  string `json:"tunnel_id"`
}

// RequestPayload represents an HTTP request framed for forwarding to CLI
type RequestPayload struct {
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Headers http.Header `json:"headers"`
	Body    []byte      `json:"body,omitempty"`
}

// ResponsePayload represents an HTTP response framed from CLI back to Gateway
type ResponsePayload struct {
	ID         string      `json:"id"`
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// ErrorPayload represents an error message
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
