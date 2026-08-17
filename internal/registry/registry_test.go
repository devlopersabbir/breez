package registry

import (
	"testing"
)

func TestRegistryOperations(t *testing.T) {
	reg := New()

	// 1. Register route
	route, err := reg.Register("myapi", "breez.local", 4000)
	if err != nil {
		t.Fatalf("unexpected error registering: %v", err)
	}

	if route.Hostname != "myapi.breez.local" {
		t.Fatalf("expected hostname myapi.breez.local, got %s", route.Hostname)
	}

	// 2. Lookup by full hostname
	foundRoute, found := reg.Lookup("myapi.breez.local")
	if !found || foundRoute.TargetPort != 4000 {
		t.Fatalf("expected to find route with port 4000")
	}

	// 3. Lookup with port suffix in host header (e.g. myapi.breez.local:80)
	foundRoute, found = reg.Lookup("myapi.breez.local:80")
	if !found || foundRoute.TargetPort != 4000 {
		t.Fatalf("expected host with port to match")
	}

	// 4. Record Hit
	reg.RecordHit("myapi.breez.local")
	if foundRoute.Requests != 1 {
		t.Fatalf("expected requests count 1, got %d", foundRoute.Requests)
	}

	// 5. Deregister
	ok := reg.Deregister("myapi.breez.local")
	if !ok {
		t.Fatalf("expected successful deregistration")
	}

	_, found = reg.Lookup("myapi.breez.local")
	if found {
		t.Fatalf("expected route to be removed")
	}
}
