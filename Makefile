.PHONY: all build test clean install uninstall start gateway serve vet fmt check-deps help

BINARY_DIR=bin
CLI_BINARY=$(BINARY_DIR)/breez
GATEWAY_BINARY=$(BINARY_DIR)/gateway

# Default target
all: build test

## build: Build both CLI and Gateway binaries into ./bin
build:
	@mkdir -p $(BINARY_DIR)
	@echo "Building Breez CLI..."
	@go build -v -o $(CLI_BINARY) ./cmd/breez
	@echo "Building Breez Gateway..."
	@go build -v -o $(GATEWAY_BINARY) ./cmd/gateway
	@echo "✔ Binaries successfully built in $(BINARY_DIR)/"

## test: Run unit and integration test suite
test:
	@echo "Running tests..."
	@go test -v -race ./...

## vet: Run static analysis / go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

## fmt: Format all Go source files
fmt:
	@echo "Formatting code..."
	@gofmt -s -w .

## check-deps: Tidy and verify go.mod dependencies
check-deps:
	@echo "Tidying go.mod..."
	@go mod tidy
	@go mod verify

## gateway: Run Gateway server locally on port 8080
gateway: build
	@echo "Starting local Gateway server on :8080..."
	@./$(GATEWAY_BINARY) --domain breez.localhost --port 8080

## serve: Run CLI tunnel serving local port 3000 (Usage: make serve PORT=3000)
PORT ?= 3000
serve: build
	@echo "Starting Breez CLI serving port $(PORT)..."
	@./$(CLI_BINARY) serve $(PORT) --gateway http://localhost:8080

INSTALL_DIR ?= /usr/local/bin

## install: Build and install breez CLI binary to $(INSTALL_DIR)
install: build
	@echo "Installing Breez CLI to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR) 2>/dev/null || true
	@cp $(CLI_BINARY) $(INSTALL_DIR)/breez 2>/dev/null || sudo cp $(CLI_BINARY) $(INSTALL_DIR)/breez
	@echo "✔ Breez CLI successfully installed to $(INSTALL_DIR)/breez"
	@echo "Run 'breez version' or 'breez --help' to verify."

## uninstall: Remove installed breez binary from $(INSTALL_DIR)
uninstall:
	@echo "Uninstalling Breez CLI from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/breez 2>/dev/null || sudo rm -f $(INSTALL_DIR)/breez
	@echo "✔ Breez CLI uninstalled."

## start: Run CLI local domain mode for port 3000 (Usage: make start PORT=3000)
start: build
	@echo "Starting Breez CLI in local mode for port $(PORT)..."
	@./$(CLI_BINARY) start $(PORT)

## clean: Remove built binaries and temporary test files
clean:
	@echo "Cleaning binaries..."
	@rm -rf $(BINARY_DIR)
	@rm -f release_notes.md
	@echo "✔ Clean completed."

## help: Show list of available Makefile commands
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':'
