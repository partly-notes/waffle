# Waffle - Well Architected Framework for Less Effort
# Build and packaging Makefile

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build configuration
BINARY_NAME = waffle
MAIN_PATH = ./cmd/waffle
BUILD_DIR = build
DIST_DIR = dist

# Go build flags
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -s -w"
GO_BUILD = go build $(LDFLAGS)

# Platforms to build for
PLATFORMS = \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

# Docker configuration
DOCKER_IMAGE = waffle
DOCKER_TAG ?= $(VERSION)

.PHONY: all
all: clean test build

.PHONY: help
help: ## Display this help message
	@echo "Waffle Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify
	@echo "Dependencies downloaded"

.PHONY: tidy
tidy: ## Tidy go.mod and go.sum
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Dependencies tidied"

.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatted"

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "Vet complete"

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint installed)
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete"

.PHONY: test-coverage
test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-property
test-property: ## Run property-based tests
	@echo "Running property-based tests..."
	@go test -v ./internal/core -run TestProperty
	@echo "Property tests complete"

.PHONY: build
build: ## Build for current platform
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	@$(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: install
install: ## Install binary to /usr/local/bin (requires sudo)
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@$(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@sudo mv $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "✓ Installed to /usr/local/bin/$(BINARY_NAME)"
	@echo "✓ Run 'waffle --version' to verify"

.PHONY: install-user
install-user: ## Install binary to $GOPATH/bin (no sudo required)
	@echo "Installing $(BINARY_NAME) to Go bin directory..."
	@$(GO_BUILD) -o $$(go env GOPATH)/bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)"
	@echo ""
	@echo "Note: Make sure $$(go env GOPATH)/bin is in your PATH"
	@echo "Add this to your ~/.zshrc or ~/.bashrc:"
	@echo '  export PATH="$$HOME/go/bin:$$PATH"'

.PHONY: build-all
build-all: clean ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	@$(foreach platform,$(PLATFORMS), \
		$(call build_platform,$(platform)) \
	)
	@echo "All builds complete"

define build_platform
	$(eval OS := $(word 1,$(subst /, ,$(1))))
	$(eval ARCH := $(word 2,$(subst /, ,$(1))))
	$(eval OUTPUT := $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-$(OS)-$(ARCH))
	$(eval EXT := $(if $(filter windows,$(OS)),.exe,))
	@echo "Building for $(OS)/$(ARCH)..."
	@GOOS=$(OS) GOARCH=$(ARCH) $(GO_BUILD) -o $(OUTPUT)$(EXT) $(MAIN_PATH)
	@if [ "$(OS)" != "windows" ]; then chmod +x $(OUTPUT)$(EXT); fi
	@echo "  -> $(OUTPUT)$(EXT)"
endef

.PHONY: release
release: clean test build-all ## Create release archives
	@echo "Creating release archives..."
	@mkdir -p $(DIST_DIR)/archives
	@for file in $(DIST_DIR)/$(BINARY_NAME)-*; do \
		if [ -f "$$file" ]; then \
			base=$$(basename $$file); \
			if echo "$$base" | grep -q "windows"; then \
				zip -j $(DIST_DIR)/archives/$$base.zip $$file README.md config.example.yaml; \
			else \
				tar -czf $(DIST_DIR)/archives/$$base.tar.gz -C $(DIST_DIR) $$(basename $$file) -C .. README.md config.example.yaml; \
			fi; \
			echo "  -> $(DIST_DIR)/archives/$$base archive"; \
		fi; \
	done
	@echo "Release archives created in $(DIST_DIR)/archives/"

.PHONY: checksums
checksums: ## Generate checksums for release files
	@echo "Generating checksums..."
	@cd $(DIST_DIR)/archives && sha256sum * > SHA256SUMS
	@echo "Checksums generated: $(DIST_DIR)/archives/SHA256SUMS"

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run --rm -it \
		-v ~/.aws:/root/.aws:ro \
		-v $(PWD):/workspace \
		-w /workspace \
		$(DOCKER_IMAGE):$(DOCKER_TAG) \
		$(ARGS)

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_IMAGE):latest
	@echo "Docker image pushed"

.PHONY: version
version: ## Display version information
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

.PHONY: info
info: ## Display build information
	@echo "Build Information:"
	@echo "  Binary Name: $(BINARY_NAME)"
	@echo "  Version:     $(VERSION)"
	@echo "  Commit:      $(COMMIT)"
	@echo "  Date:        $(DATE)"
	@echo "  Go Version:  $$(go version)"
	@echo ""
	@echo "Platforms:"
	@$(foreach platform,$(PLATFORMS), \
		echo "  - $(platform)"; \
	)

.PHONY: dev
dev: clean build ## Quick development build
	@echo "Development build complete"
	@./$(BUILD_DIR)/$(BINARY_NAME) --version

.PHONY: run
run: build ## Build and run with arguments (use ARGS="...")
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# CI/CD targets
.PHONY: ci-test
ci-test: deps vet test ## Run CI tests

.PHONY: ci-build
ci-build: deps build-all checksums ## Run CI build

.PHONY: ci-release
ci-release: ci-test ci-build release checksums ## Full CI release pipeline
