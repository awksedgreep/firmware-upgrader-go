# Firmware Upgrader - Build Configuration
# Optimized builds with UPX compression for minimal binary sizes

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS = -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

# GitHub Container Registry
GHCR_REPO = ghcr.io/awksedgreep
IMAGE_NAME = firmware-upgrader
GHCR_IMAGE = $(GHCR_REPO)/$(IMAGE_NAME)

# Binary names
BINARY_NAME = firmware-upgrader
LINUX_ARM64 = $(BINARY_NAME)-linux-arm64
LINUX_AMD64 = $(BINARY_NAME)-linux-amd64
LINUX_ARM = $(BINARY_NAME)-linux-arm
MACOS_ARM64 = $(BINARY_NAME)-darwin-arm64
MACOS_AMD64 = $(BINARY_NAME)-darwin-amd64

# Build directory
BUILD_DIR = build
CMD_DIR = cmd/firmware-upgrader

.PHONY: all clean test coverage build linux-arm64 linux-amd64 linux-arm macos help compress ghcr-login ghcr-build ghcr-push ghcr-push-minimal ghcr-release

# Default target
all: clean linux-arm64 linux-amd64 linux-arm

help: ## Show this help message
	@echo "Firmware Upgrader Build Targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build for current platform (no compression)
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

linux-arm64: ## Build optimized Linux ARM64 binary with UPX compression
	@echo "Building Linux ARM64 (optimized + compressed)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(LINUX_ARM64) ./$(CMD_DIR)
	@if command -v upx >/dev/null 2>&1; then \
		echo "Compressing with UPX..."; \
		upx --best --lzma $(BUILD_DIR)/$(LINUX_ARM64); \
	else \
		echo "UPX not found, skipping compression (install with: brew install upx)"; \
	fi
	@ls -lh $(BUILD_DIR)/$(LINUX_ARM64)

linux-amd64: ## Build optimized Linux AMD64 binary with UPX compression
	@echo "Building Linux AMD64 (optimized + compressed)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(LINUX_AMD64) ./$(CMD_DIR)
	@if command -v upx >/dev/null 2>&1; then \
		echo "Compressing with UPX..."; \
		upx --best --lzma $(BUILD_DIR)/$(LINUX_AMD64); \
	else \
		echo "UPX not found, skipping compression (install with: brew install upx)"; \
	fi
	@ls -lh $(BUILD_DIR)/$(LINUX_AMD64)

linux-arm: ## Build optimized Linux ARM (32-bit) binary with UPX compression
	@echo "Building Linux ARM (32-bit, optimized + compressed)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(LINUX_ARM) ./$(CMD_DIR)
	@if command -v upx >/dev/null 2>&1; then \
		echo "Compressing with UPX..."; \
		upx --best --lzma $(BUILD_DIR)/$(LINUX_ARM); \
	else \
		echo "UPX not found, skipping compression (install with: brew install upx)"; \
	fi
	@ls -lh $(BUILD_DIR)/$(LINUX_ARM)

macos: ## Build for macOS (current arch, no UPX due to macOS limitations)
	@echo "Building macOS binary (optimized, no compression)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)

macos-all: ## Build for both macOS architectures (no UPX)
	@echo "Building macOS ARM64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(MACOS_ARM64) ./$(CMD_DIR)
	@echo "Building macOS AMD64..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(MACOS_AMD64) ./$(CMD_DIR)
	@ls -lh $(BUILD_DIR)/$(MACOS_ARM64) $(BUILD_DIR)/$(MACOS_AMD64)

compress: ## Compress all Linux binaries in build directory with UPX
	@if command -v upx >/dev/null 2>&1; then \
		echo "Compressing all Linux binaries..."; \
		upx --best --lzma $(BUILD_DIR)/*linux* 2>/dev/null || true; \
	else \
		echo "UPX not found (install with: brew install upx)"; \
		exit 1; \
	fi

test: ## Run all tests
	@echo "Running tests..."
	go test -v ./...

coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-* firmware-upgrader-*
	@echo "Clean complete"

install-upx: ## Install UPX compression tool (macOS)
	@echo "Installing UPX..."
	@if command -v brew >/dev/null 2>&1; then \
		brew install upx; \
	else \
		echo "Homebrew not found. Please install UPX manually from: https://upx.github.io/"; \
		exit 1; \
	fi

size-compare: ## Compare sizes of all built binaries
	@echo "Binary Size Comparison:"
	@echo "======================="
	@if [ -d $(BUILD_DIR) ]; then \
		ls -lh $(BUILD_DIR)/* 2>/dev/null | awk '{printf "%-40s %10s\n", $$9, $$5}'; \
	else \
		echo "No binaries found in $(BUILD_DIR). Run 'make all' first."; \
	fi

mikrotik: ## Build optimized binaries for MikroTik routers (ARM64 + AMD64)
	@echo "Building MikroTik deployment binaries..."
	@$(MAKE) linux-arm64
	@$(MAKE) linux-amd64
	@echo ""
	@echo "MikroTik binaries ready in $(BUILD_DIR)/"
	@echo "For ARM64 routers: $(LINUX_ARM64)"
	@echo "For x86-64 routers: $(LINUX_AMD64)"

deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	@echo "Dependencies ready"

tidy: ## Tidy and verify go.mod
	@echo "Tidying dependencies..."
	go mod tidy
	go mod verify

fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...

lint: ## Run golangci-lint (requires golangci-lint installed)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install from: https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	fi

ghcr-login: ## Login to GitHub Container Registry using gh CLI and Podman
	@echo "Logging in to GitHub Container Registry..."
	@if ! command -v gh >/dev/null 2>&1; then \
		echo "ERROR: gh CLI not found. Install from: https://cli.github.com/"; \
		exit 1; \
	fi
	@if ! command -v podman >/dev/null 2>&1; then \
		echo "ERROR: podman not found. Install from: https://podman.io/"; \
		exit 1; \
	fi
	@echo $(shell gh auth token) | podman login ghcr.io -u awksedgreep --password-stdin
	@echo "✅ Logged in to GHCR as awksedgreep"

ghcr-build: ## Build container image for GHCR (multi-platform)
	@echo "Building container image for GHCR..."
	@if ! command -v podman >/dev/null 2>&1; then \
		echo "ERROR: podman not found. Install from: https://podman.io/"; \
		exit 1; \
	fi
	@echo "Image: $(GHCR_IMAGE):$(VERSION)"
	@echo "Building AMD64 image..."
	podman build --platform linux/amd64 \
		-t $(GHCR_IMAGE):$(VERSION)-amd64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		.
	@echo "Building ARM64 image..."
	podman build --platform linux/arm64 \
		-t $(GHCR_IMAGE):$(VERSION)-arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		.
	@echo "Creating manifest list..."
	podman manifest create $(GHCR_IMAGE):$(VERSION) \
		$(GHCR_IMAGE):$(VERSION)-amd64 \
		$(GHCR_IMAGE):$(VERSION)-arm64
	podman manifest create $(GHCR_IMAGE):latest \
		$(GHCR_IMAGE):$(VERSION)-amd64 \
		$(GHCR_IMAGE):$(VERSION)-arm64
	@echo "✅ Built multi-platform image: $(GHCR_IMAGE):$(VERSION)"

ghcr-push: ghcr-login ghcr-build ## Build and push container image to GHCR
	@echo "Pushing to GitHub Container Registry..."
	podman manifest push $(GHCR_IMAGE):$(VERSION) docker://$(GHCR_IMAGE):$(VERSION)
	podman manifest push $(GHCR_IMAGE):latest docker://$(GHCR_IMAGE):latest
	@echo "✅ Pushed to GHCR:"
	@echo "   $(GHCR_IMAGE):$(VERSION)"
	@echo "   $(GHCR_IMAGE):latest"
	@echo ""
	@echo "Pull with: podman pull $(GHCR_IMAGE):$(VERSION)"

ghcr-push-minimal: ghcr-login ## Build and push minimal container image to GHCR
	@echo "Building and pushing minimal image to GHCR..."
	@if ! command -v podman >/dev/null 2>&1; then \
		echo "ERROR: podman not found. Install from: https://podman.io/"; \
		exit 1; \
	fi
	@echo "Building AMD64 minimal image..."
	podman build --platform linux/amd64 \
		-f Dockerfile.minimal \
		-t $(GHCR_IMAGE):$(VERSION)-minimal-amd64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		.
	@echo "Building ARM64 minimal image..."
	podman build --platform linux/arm64 \
		-f Dockerfile.minimal \
		-t $(GHCR_IMAGE):$(VERSION)-minimal-arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		.
	@echo "Creating manifest list..."
	podman manifest create $(GHCR_IMAGE):$(VERSION)-minimal \
		$(GHCR_IMAGE):$(VERSION)-minimal-amd64 \
		$(GHCR_IMAGE):$(VERSION)-minimal-arm64
	podman manifest create $(GHCR_IMAGE):minimal \
		$(GHCR_IMAGE):$(VERSION)-minimal-amd64 \
		$(GHCR_IMAGE):$(VERSION)-minimal-arm64
	@echo "Pushing to GHCR..."
	podman manifest push $(GHCR_IMAGE):$(VERSION)-minimal docker://$(GHCR_IMAGE):$(VERSION)-minimal
	podman manifest push $(GHCR_IMAGE):minimal docker://$(GHCR_IMAGE):minimal
	@echo "✅ Pushed minimal image to GHCR:"
	@echo "   $(GHCR_IMAGE):$(VERSION)-minimal"
	@echo "   $(GHCR_IMAGE):minimal"

ghcr-release: ghcr-push ghcr-push-minimal ## Build and push all container images to GHCR (standard + minimal)
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════════╗"
	@echo "║  Successfully released to GitHub Container Registry          ║"
	@echo "╚══════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Standard images:"
	@echo "  podman pull $(GHCR_IMAGE):$(VERSION)"
	@echo "  podman pull $(GHCR_IMAGE):latest"
	@echo ""
	@echo "Minimal images:"
	@echo "  podman pull $(GHCR_IMAGE):$(VERSION)-minimal"
	@echo "  podman pull $(GHCR_IMAGE):minimal"
	@echo ""
	@echo "View at: https://github.com/awksedgreep/firmware-upgrader/pkgs/container/firmware-upgrader"

run: build ## Build and run the application
	@echo "Starting firmware-upgrader..."
	@$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run in development mode with live reload (requires air)
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not found. Install with: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi

.PHONY: check-upx
check-upx:
	@if ! command -v upx >/dev/null 2>&1; then \
		echo "WARNING: UPX not installed. Binaries will not be compressed."; \
		echo "Install with: make install-upx"; \
	fi
