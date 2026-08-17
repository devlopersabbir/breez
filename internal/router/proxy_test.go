package router

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/devlopersabbir/breez/internal/registry"
)

func TestRouterProxy(t *testing.T) {
	// 1. Create a dummy upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Echo", "hello-from-local")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mock response body"))
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	port, _ := strconv.Atoi(u.Port())

	// 2. Register route
	reg := registry.New()
	_, err := reg.Register("testapp", "breez.local", port)
	if err != nil {
		t.Fatalf("failed to register route: %v", err)
	}

	// 3. Start Router on an ephemeral high port
	routerCfg := Config{
		BindAddr: "127.0.0.1:18080",
		Domain:   "breez.local",
	}
	r := New(routerCfg, reg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = r.Start(ctx)
	}()
	time.Sleep(100 * time.Millisecond)

	// 4. Send request with Host: testapp.breez.local
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:18080/api/items", nil)
	req.Host = "testapp.breez.local"

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "mock response body" {
		t.Fatalf("expected body 'mock response body', got '%s'", string(body))
	}

	if resp.Header.Get("X-Custom-Echo") != "hello-from-local" {
		t.Fatalf("expected header X-Custom-Echo")
	}

	// 5. Test 404 for unmapped host
	req404, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:18080/", nil)
	req404.Host = "nonexistent.breez.local"
	resp404, err := client.Do(req404)
	if err != nil {
		t.Fatalf("404 request failed: %v", err)
	}
	defer resp404.Body.Close()

	if resp404.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp404.StatusCode)
	}

	cancel()
	r.Stop()
}
