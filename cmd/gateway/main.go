package main

import (
	"flag"
	"log"

	"github.com/devlopersabbir/breez/internal/gateway"
	"github.com/devlopersabbir/breez/internal/version"
)

func main() {
	domain := flag.String("domain", "breez.localhost", "Base domain for public tunnels")
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	log.Printf("Breez Gateway Version: %s (%s)", version.Version, version.Commit)

	server := gateway.NewGatewayServer(*domain, *port)
	if err := server.Start(); err != nil {
		log.Fatalf("Gateway server stopped with error: %v", err)
	}
}
