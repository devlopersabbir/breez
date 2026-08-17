package registry

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Route represents a registered local mapping from a hostname to a target port.
type Route struct {
	ID           string    `json:"id"`
	Hostname     string    `json:"hostname"`   // e.g. "myapp.breez.local"
	Subdomain    string    `json:"subdomain"`  // e.g. "myapp"
	TargetHost   string    `json:"targetHost"` // default: "127.0.0.1"
	TargetPort   int       `json:"targetPort"` // e.g. 3000
	PID          int       `json:"pid"`        // Process ID of the client
	CreatedAt    time.Time `json:"createdAt"`
	LastActiveAt time.Time `json:"lastActiveAt"`
	Requests     int64     `json:"requests"`
}

// Registry is a thread-safe registry of active routes.
type Registry struct {
	mu     sync.RWMutex
	routes map[string]*Route // normalized hostname -> route
}

// New creates an empty route registry.
func New() *Registry {
	return &Registry{
		routes: make(map[string]*Route),
	}
}

// NormalizeHost strips port and lowercases the hostname.
func NormalizeHost(host string) string {
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return strings.ToLower(strings.TrimSpace(host))
}

// Register adds or updates a route.
func (r *Registry) Register(subdomain, domain string, targetPort int) (*Route, error) {
	if targetPort <= 0 || targetPort > 65535 {
		return nil, fmt.Errorf("invalid target port %d", targetPort)
	}

	sub := strings.ToLower(strings.TrimSpace(subdomain))
	if sub == "" {
		return nil, fmt.Errorf("subdomain cannot be empty")
	}

	dom := strings.ToLower(strings.TrimSpace(domain))
	if dom == "" {
		dom = "breez.local"
	}

	hostname := fmt.Sprintf("%s.%s", sub, dom)
	now := time.Now()

	route := &Route{
		ID:           fmt.Sprintf("%s-%d", sub, now.Unix()),
		Hostname:     hostname,
		Subdomain:    sub,
		TargetHost:   "127.0.0.1",
		TargetPort:   targetPort,
		PID:          os.Getpid(),
		CreatedAt:    now,
		LastActiveAt: now,
		Requests:     0,
	}

	r.mu.Lock()
	r.routes[hostname] = route
	// Also register short subdomain if domain is breez.local for flexibility
	r.routes[sub] = route
	r.mu.Unlock()

	return route, nil
}

// Lookup finds a route by hostname or subdomain.
func (r *Registry) Lookup(host string) (*Route, bool) {
	normalized := NormalizeHost(host)
	r.mu.RLock()
	defer r.mu.RUnlock()

	route, exists := r.routes[normalized]
	return route, exists
}

// Deregister removes a route by hostname or subdomain.
func (r *Registry) Deregister(key string) bool {
	normalized := NormalizeHost(key)
	r.mu.Lock()
	defer r.mu.Unlock()

	route, exists := r.routes[normalized]
	if !exists {
		return false
	}

	delete(r.routes, route.Hostname)
	delete(r.routes, route.Subdomain)
	return true
}

// List returns all unique active routes.
func (r *Registry) List() []*Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	var list []*Route
	for _, route := range r.routes {
		if !seen[route.Hostname] {
			seen[route.Hostname] = true
			list = append(list, route)
		}
	}
	return list
}

// RecordHit increments request counter and updates last active timestamp.
func (r *Registry) RecordHit(host string) {
	normalized := NormalizeHost(host)
	r.mu.RLock()
	route, exists := r.routes[normalized]
	r.mu.RUnlock()

	if exists {
		atomic.AddInt64(&route.Requests, 1)
		route.LastActiveAt = time.Now()
	}
}
