package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/devlopersabbir/breez/internal/protocol"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type TunnelSession struct {
	ID        string
	Subdomain string
	Conn      *websocket.Conn
	ConnMu    sync.Mutex
	Pending   map[string]chan *protocol.ResponsePayload
	PendingMu sync.Mutex
}

type GatewayServer struct {
	Domain    string
	Port      int
	tunnels   map[string]*TunnelSession // subdomain -> session
	tunnelsMu sync.RWMutex
}

func NewGatewayServer(domain string, port int) *GatewayServer {
	return &GatewayServer{
		Domain:  domain,
		Port:    port,
		tunnels: make(map[string]*TunnelSession),
	}
}

func (g *GatewayServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/_breez/ws", g.handleWebSocket)
	mux.HandleFunc("/", g.handleHTTP)

	addr := fmt.Sprintf(":%d", g.Port)
	log.Printf("[Gateway] Starting server on %s (domain: %s)...", addr, g.Domain)
	return http.ListenAndServe(addr, mux)
}

func (g *GatewayServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Gateway] Upgrade error: %v", err)
		return
	}

	// Wait for init frame
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return
	}

	var frame protocol.Frame
	if err := json.Unmarshal(msgBytes, &frame); err != nil || frame.Type != protocol.MsgTunnelInit {
		conn.Close()
		return
	}

	var initPayload protocol.TunnelInitPayload
	_ = json.Unmarshal(frame.Payload, &initPayload)

	subdomain := initPayload.RequestedSubdomain
	if subdomain == "" {
		subdomain = g.generateSubdomain()
	}

	session := &TunnelSession{
		ID:        g.generateID(),
		Subdomain: subdomain,
		Conn:      conn,
		Pending:   make(map[string]chan *protocol.ResponsePayload),
	}

	g.tunnelsMu.Lock()
	g.tunnels[subdomain] = session
	g.tunnelsMu.Unlock()

	log.Printf("[Gateway] Registered tunnel '%s' (ID: %s)", subdomain, session.ID)

	// Send ready response
	readyPayload, _ := json.Marshal(protocol.TunnelReadyPayload{
		Subdomain: subdomain,
		URL:       fmt.Sprintf("http://%s.%s:%d", subdomain, g.Domain, g.Port),
		TunnelID:  session.ID,
	})

	readyFrame, _ := json.Marshal(protocol.Frame{
		Type:    protocol.MsgTunnelReady,
		Payload: readyPayload,
	})

	session.ConnMu.Lock()
	_ = conn.WriteMessage(websocket.TextMessage, readyFrame)
	session.ConnMu.Unlock()

	// Read messages loop (responses & control frames)
	defer func() {
		g.tunnelsMu.Lock()
		delete(g.tunnels, subdomain)
		g.tunnelsMu.Unlock()
		conn.Close()
		log.Printf("[Gateway] Disconnected tunnel '%s'", subdomain)
	}()

	for {
		_, bytes, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var f protocol.Frame
		if err := json.Unmarshal(bytes, &f); err != nil {
			continue
		}

		if f.Type == protocol.MsgResponse {
			var resp protocol.ResponsePayload
			if err := json.Unmarshal(f.Payload, &resp); err == nil {
				session.PendingMu.Lock()
				ch, exists := session.Pending[resp.ID]
				if exists {
					delete(session.Pending, resp.ID)
					ch <- &resp
				}
				session.PendingMu.Unlock()
			}
		}
	}
}

func (g *GatewayServer) handleHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	subdomain := ""
	if strings.HasSuffix(host, "."+g.Domain) {
		subdomain = strings.TrimSuffix(host, "."+g.Domain)
	}

	if subdomain == "" {
		http.Error(w, "Breez Gateway: Invalid Host header or direct access", http.StatusBadRequest)
		return
	}

	g.tunnelsMu.RLock()
	session, exists := g.tunnels[subdomain]
	g.tunnelsMu.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("Breez Gateway: Tunnel '%s' not found or offline", subdomain), http.StatusNotFound)
		return
	}

	reqID := g.generateID()
	body, _ := io.ReadAll(r.Body)

	reqPayload, _ := json.Marshal(protocol.RequestPayload{
		ID:      reqID,
		Method:  r.Method,
		Path:    r.URL.RequestURI(),
		Headers: r.Header,
		Body:    body,
	})

	reqFrame, _ := json.Marshal(protocol.Frame{
		Type:    protocol.MsgRequest,
		ID:      reqID,
		Payload: reqPayload,
	})

	respCh := make(chan *protocol.ResponsePayload, 1)

	session.PendingMu.Lock()
	session.Pending[reqID] = respCh
	session.PendingMu.Unlock()

	session.ConnMu.Lock()
	err := session.Conn.WriteMessage(websocket.TextMessage, reqFrame)
	session.ConnMu.Unlock()

	if err != nil {
		session.PendingMu.Lock()
		delete(session.Pending, reqID)
		session.PendingMu.Unlock()
		http.Error(w, "Breez Gateway: Failed to forward request to client", http.StatusBadGateway)
		return
	}

	select {
	case resp := <-respCh:
		for k, values := range resp.Headers {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(resp.Body)
	case <-time.After(30 * time.Second):
		session.PendingMu.Lock()
		delete(session.Pending, reqID)
		session.PendingMu.Unlock()
		http.Error(w, "Breez Gateway: Tunnel response timeout", http.StatusGatewayTimeout)
	}
}

func (g *GatewayServer) generateSubdomain() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (g *GatewayServer) generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
