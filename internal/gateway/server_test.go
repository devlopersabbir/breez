package gateway

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/devlopersabbir/breez/internal/cli"
)

func TestEndToEndTunneling(t *testing.T) {
	// 1. Create dummy local server
	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Hello from local server!"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer localServer.Close()

	// Extract local port
	parts := strings.Split(localServer.URL, ":")
	portStr := parts[len(parts)-1]
	var localPort int
	for _, ch := range portStr {
		localPort = localPort*10 + int(ch-'0')
	}

	// 2. Create Gateway Server
	gwServer := NewGatewayServer("breez.localhost", 8080)
	mux := http.NewServeMux()
	mux.HandleFunc("/_breez/ws", gwServer.handleWebSocket)
	mux.HandleFunc("/", gwServer.handleHTTP)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 3. Connect CLI in background
	client := cli.NewClient(ts.URL, localPort, "testsub")
	done := make(chan error, 1)
	go func() {
		done <- client.Serve()
	}()

	// Give handshake time to finish
	for i := 0; i < 50; i++ {
		gwServer.tunnelsMu.RLock()
		_, exists := gwServer.tunnels["testsub"]
		gwServer.tunnelsMu.RUnlock()
		if exists {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 4. Send HTTP request to Gateway for 'testsub'
	req, err := http.NewRequest("GET", ts.URL+"/hello", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Host = "testsub.breez.localhost"

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	if !bytes.Equal(body, []byte("Hello from local server!")) {
		t.Fatalf("unexpected body: %s", string(body))
	}

	// Clean shutdown gateway tunnel connection to let client.Serve() exit gracefully
	gwServer.tunnelsMu.Lock()
	if session, exists := gwServer.tunnels["testsub"]; exists {
		session.Conn.Close()
	}
	gwServer.tunnelsMu.Unlock()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
	}
}
