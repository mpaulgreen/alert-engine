# Alert Engine Makefile - Initial version generated - NOT TESTED
# Provides convenient commands for building, testing, and deploying

# Configuration
PROJECT_NAME := alert-engine
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
REGISTRY ?= quay.io/mpaulgreen
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

##@ OpenShift Infrastructure

.PHONY: infra-setup
infra-setup: ## Setup complete OpenShift infrastructure (Kafka, Redis, Logging)
	@echo "$(BLUE)Setting up OpenShift infrastructure for Alert Engine...$(NC)"
	@echo "$(YELLOW)This will deploy Kafka, Redis Cluster, and OpenShift Logging$(NC)"
	./scripts/setup_openshift_infrastructure.sh

.PHONY: infra-validate
infra-validate: ## Validate OpenShift infrastructure deployment
	@echo "$(BLUE)Validating OpenShift infrastructure...$(NC)"
	./scripts/validate_openshift_infrastructure.sh

.PHONY: infra-verify-cleanup
infra-verify-cleanup: ## Verify resources before cleanup (inventory check)
	@echo "$(BLUE)Verifying resources before cleanup...$(NC)"
	./scripts/verify_resources_before_cleanup.sh

.PHONY: infra-cleanup
infra-cleanup: ## Cleanup OpenShift infrastructure
	@echo "$(YELLOW)WARNING: This will delete all infrastructure components!$(NC)"
	@echo "$(YELLOW)Press Ctrl+C to cancel, or wait 10 seconds to continue...$(NC)"
	@sleep 10
	./scripts/cleanup_openshift_infrastructure.sh

.PHONY: infra-status
infra-status: ## Show quick status of all infrastructure components
	@echo "$(BLUE)OpenShift Infrastructure Status$(NC)"
	@echo "================================"
	@echo "$(GREEN)Kafka Cluster:$(NC)"
	@kubectl get kafka alert-kafka-cluster -n amq-streams-kafka -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null && echo " (Ready)" || echo " (Not Found/Not Ready)"
	@echo "$(GREEN)Redis Cluster:$(NC)"
	@kubectl get pods -l app=redis-cluster -n redis-cluster --no-headers 2>/dev/null | grep Running | wc -l | xargs -I {} echo "{}/6 pods running"
	@echo "$(GREEN)ClusterLogForwarder:$(NC)"
	@kubectl get clusterlogforwarder kafka-forwarder -n openshift-logging 2>/dev/null && echo " (Deployed)" || echo " (Not Found)"
	@echo "$(GREEN)Alert Engine Namespace:$(NC)"
	@kubectl get namespace alert-engine 2>/dev/null && echo " (Exists)" || echo " (Not Found)"

##@ Testing

.PHONY: test
test: ## Run unit tests
	@echo "$(BLUE)Running unit tests...$(NC)"
	go test -v -tags=unit ./internal/... ./pkg/...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	go test -v -tags=unit -coverprofile=coverage.out ./internal/... ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	go test -v -tags=integration ./internal/... ./pkg/...

.PHONY: test-all
test-all: ## Run all tests (unit + integration)
	@echo "$(BLUE)Running all tests...$(NC)"
	go test -v -tags=unit,integration ./internal/... ./pkg/...

##@ Advanced Testing (using scripts)

.PHONY: test-unit-scripts
test-unit-scripts: ## Run unit tests using advanced script (with per-package analysis)
	@echo "$(BLUE)Running unit tests with advanced script...$(NC)"
	./scripts/run_unit_tests.sh

.PHONY: test-unit-coverage-scripts
test-unit-coverage-scripts: ## Run unit tests with detailed coverage using script
	@echo "$(BLUE)Running unit tests with detailed coverage analysis...$(NC)"
	./scripts/run_unit_tests.sh --coverage

.PHONY: test-integration-scripts
test-integration-scripts: ## Run integration tests using advanced script (with container management)
	@echo "$(BLUE)Running integration tests with container management...$(NC)"
	SKIP_HEALTH_CHECK=true ./scripts/run_integration_tests.sh

.PHONY: test-kafka-scripts
test-kafka-scripts: ## Run Kafka integration tests using specialized script
	@echo "$(BLUE)Running Kafka integration tests with advanced options...$(NC)"
	./scripts/run_kafka_integration_tests.sh

.PHONY: test-kafka-race
test-kafka-race: ## Run Kafka integration tests with race detection
	@echo "$(BLUE)Running Kafka tests with race detection...$(NC)"
	./scripts/run_kafka_integration_tests.sh -m race-safe -r

.PHONY: test-all-scripts
test-all-scripts: ## Run all tests using advanced scripts (comprehensive testing)
	@echo "$(BLUE)Running comprehensive test suite with scripts...$(NC)"
	./scripts/run_unit_tests.sh --coverage
	SKIP_HEALTH_CHECK=true ./scripts/run_integration_tests.sh

##@ End-to-End Testing

.PHONY: test-e2e-local
test-e2e-local: build ## Run local end-to-end tests (teardown, setup, start server, run tests)
	@echo "$(BLUE)Running local end-to-end test suite...$(NC)"
	@echo "$(YELLOW)Step 1: Tearing down existing e2e environment...$(NC)"
	./local_e2e/setup/teardown_local_e2e.sh
	@echo "$(YELLOW)Step 2: Setting up fresh e2e environment...$(NC)"
	./local_e2e/setup/setup_local_e2e.sh
	@echo "$(YELLOW)Step 3: Starting alert engine server in background...$(NC)"
	@./local_e2e/setup/start_alert_engine.sh > /tmp/alert-engine-e2e.log 2>&1 & echo $$! > /tmp/alert-engine-e2e.pid
	@echo "$(YELLOW)Step 4: Waiting for server to start...$(NC)"
	@sleep 15
	@echo "$(YELLOW)Step 5: Running e2e tests...$(NC)"
	@./local_e2e/tests/run_e2e_tests.sh || (echo "$(RED)E2E tests failed, stopping server...$(NC)" && kill `cat /tmp/alert-engine-e2e.pid` 2>/dev/null; exit 1)
	@echo "$(YELLOW)Step 6: Stopping alert engine server...$(NC)"
	@kill `cat /tmp/alert-engine-e2e.pid` 2>/dev/null || echo "Server already stopped"
	@rm -f /tmp/alert-engine-e2e.pid /tmp/alert-engine-e2e.log
	@echo "$(GREEN)Local e2e test suite completed!$(NC)"

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
	./deployments/alert-engine/build.sh --version $(VERSION) --registry $(REGISTRY)

.PHONY: docker-push
docker-push: ## Build and push container image
	@echo "$(BLUE)Building and pushing container image...$(NC)"
	./deployments/alert-engine/build.sh --version $(VERSION) --registry $(REGISTRY) --push

.PHONY: docker-test
docker-test: ## Build container image with tests
	@echo "$(BLUE)Building container image with tests...$(NC)"
	./deployments/alert-engine/build.sh --version $(VERSION) --registry $(REGISTRY) --test

##@ OpenShift Deployment

.PHONY: deploy
deploy: ## Deploy to OpenShift/Kubernetes
	@echo "$(BLUE)Deploying Alert Engine...$(NC)"
	oc apply -k deployments/alert-engine/

.PHONY: build-and-deploy
build-and-deploy: docker-push deploy ## Build, push, and deploy with custom image
	@echo "$(GREEN)Build and deployment pipeline completed!$(NC)"

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
		echo "$(BLUE)Pod: $$POD$(NC)"; \
		echo "$(BLUE)Testing health endpoint via port-forward...$(NC)"; \
		(oc port-forward pod/$$POD 18080:8080 -n alert-engine >/dev/null 2>&1 &) && \
		PORTFW_PID=$$! && \
		sleep 2 && \
		HEALTH_RESPONSE=$$(curl -s --max-time 5 http://localhost:18080/api/v1/health 2>/dev/null || echo "{}") && \
		kill $$PORTFW_PID 2>/dev/null || true && \
		wait $$PORTFW_PID 2>/dev/null || true && \
		if echo "$$HEALTH_RESPONSE" | grep -q '"success":true'; then \
			echo "$(GREEN)✅ Alert Engine is healthy$(NC)"; \
			echo "$$HEALTH_RESPONSE"; \
		else \
			echo "$(RED)❌ Alert Engine health check failed$(NC)"; \
			echo "Response: $$HEALTH_RESPONSE"; \
			exit 1; \
		fi; \
	else \
		echo "$(YELLOW)No Alert Engine pods found$(NC)"; \
		exit 1; \
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