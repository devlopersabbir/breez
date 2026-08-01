package main

import (
	"fmt"
	"os"

	"github.com/devlopersabbir/breez/internal/version"
)

func main() {
	fmt.Printf("Breez Project Base (CLI v%s)\n", version.Version)
	fmt.Println("Run 'go run ./cmd/breez serve <port>' or 'go run ./cmd/gateway' to start services.")
	os.Exit(0)
}
