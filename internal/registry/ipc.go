package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	DefaultControlTCP = "127.0.0.1:7744"
)

// GetSocketPath returns the path to the Breez IPC Unix socket.
func GetSocketPath() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	dir := filepath.Join(home, ".breez")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "breez.sock")
}

// IPCServer provides an HTTP-based control endpoint over Unix socket or TCP loopback.
type IPCServer struct {
	reg      *Registry
	listener net.Listener
	httpSrv  *http.Server
}

// RegisterRequest payload for registering a route via IPC.
type RegisterRequest struct {
	Subdomain  string `json:"subdomain"`
	Domain     string `json:"domain"`
	TargetPort int    `json:"targetPort"`
}

// StartIPCServer starts the IPC control server.
func StartIPCServer(reg *Registry) (*IPCServer, error) {
	sockPath := GetSocketPath()
	var l net.Listener
	var err error

	if sockPath != "" {
		_ = os.Remove(sockPath)
		l, err = net.Listen("unix", sockPath)
		if err != nil {
			// Fallback to TCP loopback
			l, err = net.Listen("tcp", DefaultControlTCP)
		}
	} else {
		l, err = net.Listen("tcp", DefaultControlTCP)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to start IPC listener: %w", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/routes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			routes := reg.List()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(routes)

		case http.MethodPost:
			var req RegisterRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			route, err := reg.Register(req.Subdomain, req.Domain, req.TargetPort)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(route)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/routes/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			key := strings.TrimPrefix(r.URL.Path, "/routes/")
			if reg.Deregister(key) {
				w.WriteHeader(http.StatusNoContent)
			} else {
				http.Error(w, "Route not found", http.StatusNotFound)
			}
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	httpSrv := &http.Server{Handler: mux}

	go func() {
		_ = httpSrv.Serve(l)
	}()

	return &IPCServer{
		reg:      reg,
		listener: l,
		httpSrv:  httpSrv,
	}, nil
}

// Stop shuts down the IPC server.
func (s *IPCServer) Stop() {
	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = s.httpSrv.Shutdown(ctx)
	}
	sockPath := GetSocketPath()
	if sockPath != "" {
		_ = os.Remove(sockPath)
	}
}

// IPCClient allows other CLI commands to interact with the running daemon/router.
type IPCClient struct {
	client *http.Client
}

// NewIPCClient returns an IPC client connected via Unix socket or fallback TCP.
func NewIPCClient() *IPCClient {
	sockPath := GetSocketPath()

	transport := &http.Transport{
		DialContext: func(ctx context.Context, proto, addr string) (net.Conn, error) {
			if sockPath != "" {
				if _, err := os.Stat(sockPath); err == nil {
					return net.Dial("unix", sockPath)
				}
			}
			return net.Dial("tcp", DefaultControlTCP)
		},
	}

	return &IPCClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   2 * time.Second,
		},
	}
}

// Ping checks if the local Breez daemon is running.
func (c *IPCClient) Ping() bool {
	resp, err := c.client.Get("http://daemon/ping")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// RegisterRoute registers a route on the running daemon.
func (c *IPCClient) RegisterRoute(subdomain, domain string, targetPort int) (*Route, error) {
	reqBody, _ := json.Marshal(RegisterRequest{
		Subdomain:  subdomain,
		Domain:     domain,
		TargetPort: targetPort,
	})

	resp, err := c.client.Post("http://daemon/routes", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to contact daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed (%d): %s", resp.StatusCode, string(body))
	}

	var route Route
	if err := json.NewDecoder(resp.Body).Decode(&route); err != nil {
		return nil, err
	}
	return &route, nil
}

// DeregisterRoute removes a route on the running daemon.
func (c *IPCClient) DeregisterRoute(key string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://daemon/routes/%s", key), nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deregistration returned status %d", resp.StatusCode)
	}
	return nil
}

// ListRoutes fetches all active routes from the running daemon.
func (c *IPCClient) ListRoutes() ([]*Route, error) {
	resp, err := c.client.Get("http://daemon/routes")
	if err != nil {
		return nil, fmt.Errorf("failed to contact daemon: %w", err)
	}
	defer resp.Body.Close()

	var routes []*Route
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return nil, err
	}
	return routes, nil
}
