.PHONY: build install uninstall clean test run help

BINARY_NAME=lazyhelm
INSTALL_DIR=/usr/local/bin
GO=go
GOFLAGS=-v

help: ## Show this help
	@echo "LazyHelm - Makefile commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/lazyhelm

install: build ## Build and install to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@if [ -w "$(INSTALL_DIR)" ]; then \
		mv $(BINARY_NAME) $(INSTALL_DIR)/; \
	else \
		sudo mv $(BINARY_NAME) $(INSTALL_DIR)/; \
	fi
	@echo "✓ Installed successfully!"

uninstall: ## Remove the binary from /usr/local/bin
	@echo "Uninstalling $(BINARY_NAME)..."
	@if [ -w "$(INSTALL_DIR)" ]; then \
		rm -f $(INSTALL_DIR)/$(BINARY_NAME); \
	else \
		sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME); \
	fi
	@echo "✓ Uninstalled successfully!"

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@go clean

test: ## Run tests
	@echo "Running tests..."
	$(GO) test ./...

run: build ## Build and run
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

dev: ## Run without building (go run)
	@echo "Running in development mode..."
	$(GO) run ./cmd/lazyhelm/main.go

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

release: ## Build for multiple platforms
	@echo "Building releases..."
	@mkdir -p dist
	GOOS=darwin GOARCH=amd64 $(GO) build -o dist/$(BINARY_NAME)_darwin_amd64 ./cmd/lazyhelm
	GOOS=darwin GOARCH=arm64 $(GO) build -o dist/$(BINARY_NAME)_darwin_arm64 ./cmd/lazyhelm
	GOOS=linux GOARCH=amd64 $(GO) build -o dist/$(BINARY_NAME)_linux_amd64 ./cmd/lazyhelm
	GOOS=linux GOARCH=arm64 $(GO) build -o dist/$(BINARY_NAME)_linux_arm64 ./cmd/lazyhelm
	@echo "✓ Release binaries created in dist/"
