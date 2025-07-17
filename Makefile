# Alert Engine Makefile - Initial version generated - NOT TESTED
# Provides convenient commands for building, testing, and deploying

# Configuration
PROJECT_NAME := alert-engine
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
REGISTRY ?= quay.io/alert-engine
IMAGE_NAME ?= alert-engine
FULL_IMAGE := $(REGISTRY)/$(IMAGE_NAME):$(VERSION)

# Go configuration
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED ?= 0
GOLANGCI_LINT ?= $(shell go env GOPATH)/bin/golangci-lint

# Build flags
LDFLAGS := -w -s -X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_FLAGS := -a -installsuffix cgo -ldflags="$(LDFLAGS)"

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m

# Default target
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)Alert Engine Build Commands$(NC)"
	@echo "=============================="
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(GREEN)<target>$(NC)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Format Go code
	@echo "$(BLUE)Formatting Go code...$(NC)"
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint to be installed)
	@echo "$(BLUE)Running golangci-lint...$(NC)"
	$(GOLANGCI_LINT) run

.PHONY: tidy
tidy: ## Tidy Go modules
	@echo "$(BLUE)Tidying Go modules...$(NC)"
	go mod tidy

##@ Testing

.PHONY: test
test: ## Run unit tests
	@echo "$(BLUE)Running unit tests...$(NC)"
	go test -v ./internal/... ./pkg/...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	go test -v -coverprofile=coverage.out ./internal/... ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	go test -v -tags=integration ./...

.PHONY: test-all
test-all: test test-integration ## Run all tests

##@ Building

.PHONY: build
build: ## Build the binary locally
	@echo "$(BLUE)Building Alert Engine binary...$(NC)"
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(BUILD_FLAGS) -o alert-engine ./cmd/server

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -f alert-engine
	rm -f coverage.out coverage.html
	go clean -cache

##@ Container Operations

.PHONY: docker-build
docker-build: ## Build container image
	@echo "$(BLUE)Building container image...$(NC)"
	./deployments/alert-engine/build.sh --version $(VERSION)

.PHONY: docker-push
docker-push: ## Build and push container image
	@echo "$(BLUE)Building and pushing container image...$(NC)"
	./deployments/alert-engine/build.sh --version $(VERSION) --push

.PHONY: docker-test
docker-test: ## Build container image with tests
	@echo "$(BLUE)Building container image with tests...$(NC)"
	./deployments/alert-engine/build.sh --version $(VERSION) --test

##@ OpenShift Deployment

.PHONY: deploy-local
deploy-local: ## Deploy to local OpenShift/Kubernetes
	@echo "$(BLUE)Deploying to local cluster...$(NC)"
	oc apply -k deployments/alert-engine/

.PHONY: deploy-staging
deploy-staging: docker-push ## Build, push, and deploy to staging
	@echo "$(BLUE)Deploying to staging...$(NC)"
	oc apply -k deployments/alert-engine/

.PHONY: logs
logs: ## Show logs from deployed pods
	@echo "$(BLUE)Showing Alert Engine logs...$(NC)"
	oc logs -n alert-engine -l app.kubernetes.io/name=alert-engine -f

.PHONY: status
status: ## Show deployment status
	@echo "$(BLUE)Checking deployment status...$(NC)"
	oc get all -n alert-engine

.PHONY: health
health: ## Check health of deployed Alert Engine
	@echo "$(BLUE)Checking Alert Engine health...$(NC)"
	@POD=$$(oc get pods -n alert-engine -l app.kubernetes.io/name=alert-engine -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) && \
	if [ -n "$$POD" ]; then \
		oc exec -n alert-engine $$POD -- /app/alert-engine --health-check; \
	else \
		echo "$(YELLOW)No Alert Engine pods found$(NC)"; \
	fi

##@ Utilities

.PHONY: deps
deps: ## Download dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	go mod download

.PHONY: update-deps
update-deps: ## Update dependencies
	@echo "$(BLUE)Updating dependencies...$(NC)"
	go get -u ./...
	go mod tidy

.PHONY: verify
verify: fmt vet test ## Run all verification checks

.PHONY: all
all: verify build ## Run all checks and build

##@ Development Workflow

.PHONY: dev-setup
dev-setup: deps ## Set up development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@echo "$(GREEN)Development environment ready!$(NC)"
	@echo "Try: make verify"

.PHONY: dev-test
dev-test: ## Quick development test (fmt, vet, test)
	@echo "$(BLUE)Running development checks...$(NC)"
	@$(MAKE) fmt vet test

.PHONY: release-build
release-build: clean verify docker-test docker-push ## Full release build with all checks

##@ Information

.PHONY: version
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Image: $(FULL_IMAGE)"
	@echo "GOOS: $(GOOS)"
	@echo "GOARCH: $(GOARCH)"

.PHONY: info
info: version ## Show build information
	@echo "Project: $(PROJECT_NAME)"
	@echo "Registry: $(REGISTRY)"
	@echo "Go version: $$(go version)"
	@echo "Build flags: $(BUILD_FLAGS)" 