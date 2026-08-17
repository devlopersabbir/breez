package router

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/devlopersabbir/breez/internal/registry"
)

// Config holds router configuration.
type Config struct {
	BindAddr string // e.g. "127.0.0.1:80" or "127.0.0.1:8080"
	Domain   string // default: "breez.local"
}

// Router is a local HTTP reverse proxy routing incoming requests based on Host headers.
type Router struct {
	cfg      Config
	reg      *registry.Registry
	server   *http.Server
	listener net.Listener
	mu       sync.Mutex
	running  bool
}

// New creates a new local HTTP router.
func New(cfg Config, reg *registry.Registry) *Router {
	if cfg.BindAddr == "" {
		cfg.BindAddr = "127.0.0.1:80"
	}
	if cfg.Domain == "" {
		cfg.Domain = "breez.local"
	}
	return &Router{
		cfg: cfg,
		reg: reg,
	}
}

// Start starts the HTTP router listener.
func (r *Router) Start(ctx context.Context) error {
	l, err := net.Listen("tcp", r.cfg.BindAddr)
	if err != nil {
		return fmt.Errorf("failed to bind router on %s: %w", r.cfg.BindAddr, err)
	}

	r.mu.Lock()
	r.listener = l
	r.server = &http.Server{
		Handler: r,
	}
	r.running = true
	r.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		if err := r.server.Serve(l); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		r.Stop()
		return nil
	case err := <-errCh:
		r.Stop()
		return err
	}
}

// Stop shuts down the router server.
func (r *Router) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return
	}
	r.running = false
	if r.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = r.server.Shutdown(ctx)
	}
}

// ServeHTTP inspects the Host header and proxies the request to the mapped target port.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route, found := r.reg.Lookup(req.Host)
	if !found {
		r.renderNotFound(w, req)
		return
	}

	r.reg.RecordHit(req.Host)

	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%d", route.TargetHost, route.TargetPort))
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid target URL: %v", err), http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	originalDirector := proxy.Director

	proxy.Director = func(outReq *http.Request) {
		originalDirector(outReq)
		outReq.Header.Set("X-Forwarded-Host", req.Host)
		outReq.Header.Set("X-Forwarded-Proto", "http")
		outReq.Header.Set("X-Forwarded-For", req.RemoteAddr)
		outReq.Host = req.Host
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, proxyErr error) {
		log.Printf("[Router] Proxy error for %s -> :%d: %v", req.Host, route.TargetPort, proxyErr)
		rw.WriteHeader(http.StatusBadGateway)
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(rw, `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>502 Bad Gateway - Breez</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0f172a; color: #f8fafc; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
    .card { background: #1e293b; padding: 2rem 3rem; border-radius: 12px; box-shadow: 0 10px 25px rgba(0,0,0,0.5); max-width: 500px; text-align: center; border: 1px solid #334155; }
    h1 { color: #f43f5e; margin-top: 0; }
    p { color: #94a3b8; line-height: 1.5; }
    .badge { background: #334155; color: #38bdf8; padding: 4px 8px; border-radius: 4px; font-family: monospace; }
  </style>
</head>
<body>
  <div class="card">
    <h1>☁ 502 Bad Gateway</h1>
    <p>Breez Local Router could not connect to your local application running on <span class="badge">localhost:%d</span>.</p>
    <p>Please make sure your development server on port <strong>%d</strong> is active and running.</p>
  </div>
</body>
</html>`, route.TargetPort, route.TargetPort)
	}

	proxy.ServeHTTP(w, req)
}

const notFoundTemplate = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>404 Route Not Found - Breez</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #090d16; color: #f8fafc; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; }
    .card { background: #131b2e; padding: 2.5rem; border-radius: 16px; box-shadow: 0 20px 40px rgba(0,0,0,0.6); max-width: 560px; width: 100%; border: 1px solid #1e293b; }
    .logo { display: inline-flex; align-items: center; gap: 8px; font-size: 1.4rem; font-weight: 700; color: #38bdf8; margin-bottom: 1.5rem; }
    h1 { font-size: 1.6rem; color: #f1f5f9; margin-top: 0; margin-bottom: 0.5rem; }
    p { color: #94a3b8; font-size: 0.95rem; line-height: 1.6; margin-bottom: 1.5rem; }
    .target { color: #fbbf24; font-family: monospace; font-weight: bold; }
    .active-list { background: #0a0f1d; border-radius: 8px; border: 1px solid #1e293b; padding: 1rem; margin-top: 1rem; }
    .route-item { display: flex; justify-content: space-between; align-items: center; padding: 0.5rem 0; border-bottom: 1px solid #1e293b; }
    .route-item:last-child { border-bottom: none; }
    .route-link { color: #38bdf8; text-decoration: none; font-weight: 600; font-family: monospace; }
    .route-link:hover { text-decoration: underline; }
    .port-badge { background: #1e293b; color: #a5b4fc; padding: 2px 8px; border-radius: 4px; font-size: 0.8rem; font-family: monospace; }
    .footer { margin-top: 2rem; font-size: 0.8rem; color: #64748b; text-align: center; }
  </style>
</head>
<body>
  <div class="card">
    <div class="logo">☁ Breez Local Router</div>
    <h1>404 — Route Not Found</h1>
    <p>No active tunnel or local route matches <span class="target">{{.RequestedHost}}</span>.</p>
    
    {{if .Routes}}
      <p style="margin-bottom: 0.5rem; font-weight: 600; color: #cbd5e1;">Active Local Routes:</p>
      <div class="active-list">
        {{range .Routes}}
          <div class="route-item">
            <a class="route-link" href="http://{{.Hostname}}">http://{{.Hostname}}</a>
            <span class="port-badge">➜ :{{.TargetPort}}</span>
          </div>
        {{end}}
      </div>
    {{else}}
      <p style="color: #64748b; font-style: italic;">No other local routes are currently registered.<br>Start one with: <code>breez start 3000 --name myapp</code></p>
    {{end}}

    <div class="footer">Breez Development Platform &bull; Local DNS & Router</div>
  </div>
</body>
</html>`

type notFoundData struct {
	RequestedHost string
	Routes        []*registry.Route
}

func (r *Router) renderNotFound(w http.ResponseWriter, req *http.Request) {
	tmpl, err := template.New("404").Parse(notFoundTemplate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Breez Router: 404 Route '%s' Not Found", req.Host), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, notFoundData{
		RequestedHost: req.Host,
		Routes:        r.reg.List(),
	})
}
