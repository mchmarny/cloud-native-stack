# Makefile for the cloud-native-stack project
# Purpose: Build, lint, test, and manage releases for the cloud-native-stack project.

REPO_NAME          := cloud-native-stack
VERSION            ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
IMAGE_REGISTRY     ?= ghcr.io/nvidia
IMAGE_TAG          ?= latest
YAML_FILES         := $(shell find . -type f \( -iname "*.yml" -o -iname "*.yaml" \) ! -path "./examples/*" ! -path "./~archive/*" ! -path "./bundles/*" ! -path "./.flox/*")
COMMIT             := $(shell git rev-parse HEAD)
SHORT_COMMIT       := $(shell git rev-parse --short HEAD)
DATE               := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BRANCH             := $(shell git rev-parse --abbrev-ref HEAD)
GO_VERSION         := $(shell go env GOVERSION 2>/dev/null | sed 's/go//')
GOLINT_VERSION      = $(shell golangci-lint --version 2>/dev/null | awk '{print $$4}' | sed 's/golangci-lint version //' || echo "not installed")
KO_VERSION          = $(shell ko version 2>/dev/null || echo "not installed")
GORELEASER_VERSION  = $(shell goreleaser --version 2>/dev/null | sed -n 's/^GitVersion:[[:space:]]*//p' || echo "not installed")
COVERAGE_THRESHOLD ?= 70

# Default target
all: help

.PHONY: info
info: ## Prints the current project info
	@echo "version:        $(VERSION)"
	@echo "commit:         $(COMMIT)"
	@echo "branch:         $(BRANCH)"
	@echo "repo:           $(REPO_NAME)"
	@echo "go:             $(GO_VERSION)"
	@echo "linter:         $(GOLINT_VERSION)"
	@echo "ko:             $(KO_VERSION)"
	@echo "goreleaser:     $(GORELEASER_VERSION)"

.PHONY: check-tools
check-tools: ## Verifies required tools are installed
	@echo "Checking required tools..."
	@command -v golangci-lint >/dev/null || (echo "ERROR: golangci-lint not installed" && exit 1)
	@command -v yamllint >/dev/null || (echo "ERROR: yamllint not installed" && exit 1)
	@command -v grype >/dev/null || (echo "ERROR: grype not installed" && exit 1)
	@command -v goreleaser >/dev/null || (echo "ERROR: goreleaser not installed" && exit 1)
	@command -v ko >/dev/null || (echo "ERROR: ko not installed" && exit 1)
	@echo "All required tools installed"

.PHONY: deps
deps: ## Installs required development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/goreleaser/goreleaser/v2@latest
	@go install github.com/google/ko@latest
	@go install github.com/anchore/grype@latest
	@pip install --user yamllint 2>/dev/null || echo "Install yamllint manually: pip install yamllint"
	@echo "Development tools installed"

.PHONY: tidy
tidy: ## Formats code and updates Go module dependencies
	@set -e; \
	go fmt ./...; \
	go mod tidy

.PHONY: fmt-check
fmt-check: ## Checks if code is formatted (CI-friendly, no modifications)
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make tidy' to fix:" && gofmt -l . && exit 1)
	@echo "Code formatting check passed"

.PHONY: upgrade
upgrade: ## Upgrades all dependencies to latest versions
	@set -e; \
	go get -u ./...; \
	go mod tidy

.PHONY: generate
generate: ## Runs go generate for code generation
	@echo "Running go generate..."
	@go generate ./...
	@echo "Code generation completed"

.PHONY: lint
lint: lint-go lint-yaml ## Lints the entire project (Go and YAML)
	@echo "Completed Go and YAML lints"

.PHONY: lint-go
lint-go: ## Lints Go files with golangci-lint and go vet
	@set -e; \
	echo "Running go vet..."; \
	go vet ./...; \
	echo "Running golangci-lint..."; \
	golangci-lint -c .golangci.yaml run

.PHONY: lint-yaml
lint-yaml: ## Lints YAML files with yamllint
	@if [ -n "$(YAML_FILES)" ]; then \
		yamllint -c .yamllint.yaml $(YAML_FILES); \
	else \
		echo "No YAML files found to lint."; \
	fi

.PHONY: test
test: ## Runs unit tests with race detector and coverage
	@set -e; \
	echo "Running tests with race detector..."; \
	go test -count=1 -race -covermode=atomic -coverprofile=coverage.out ./... || exit 1; \
	echo "Test coverage:"; \
	go tool cover -func=coverage.out | tail -1

.PHONY: test-coverage
test-coverage: test ## Runs tests and enforces coverage threshold (COVERAGE_THRESHOLD=60)
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$coverage% (threshold: $(COVERAGE_THRESHOLD)%)"; \
	if [ $$(echo "$$coverage < $(COVERAGE_THRESHOLD)" | bc) -eq 1 ]; then \
		echo "ERROR: Coverage $$coverage% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi; \
	echo "Coverage check passed"

.PHONY: bench
bench: ## Runs benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

.PHONY: e2e
e2e: ## Runs end-to-end integration tests
	@set -e; \
	echo "Running e2e integration tests..."; \
	tools/e2e

.PHONY: scan
scan: ## Scans for vulnerabilities with grype
	@set -e; \
	echo "Running vulnerability scan..."; \
	grype dir:. --config .grype.yaml --fail-on high --quiet

.PHONY: qualify
qualify: test lint e2e scan ## Qualifies the codebase (test, lint, e2e, scan)
	@echo "Codebase qualification completed"

.PHONY: install
install: ## Installs cnsctl binary to GOPATH/bin
	@echo "Installing cnsctl $(VERSION)..."
	@go install -ldflags "-X github.com/NVIDIA/cloud-native-stack/pkg/cli.version=$(VERSION) -X github.com/NVIDIA/cloud-native-stack/pkg/cli.commit=$(SHORT_COMMIT) -X github.com/NVIDIA/cloud-native-stack/pkg/cli.date=$(DATE)" ./cmd/cnsctl
	@echo "Installed cnsctl to $$(go env GOPATH)/bin/cnsctl"

.PHONY: server
server: ## Starts a local development server with debug logging
	@set -e; \
	echo "Starting local development server..."; \
	LOG_LEVEL=debug go run cmd/cnsd/main.go

.PHONY: docs
docs: ## Serves Go documentation on http://localhost:6060
	@set -e; \
	echo "Starting Go documentation server on http://localhost:6060"; \
	command -v pkgsite >/dev/null 2>&1 && pkgsite -http=:6060 || \
	(command -v godoc >/dev/null 2>&1 && godoc -http=:6060 || \
	(echo "Installing pkgsite..." && go install golang.org/x/pkgsite/cmd/pkgsite@latest && pkgsite -http=:6060))

.PHONY: build
build: tidy ## Builds binaries for the current OS and architecture
	@set -e; \
	goreleaser build --clean --single-target --snapshot --timeout 10m0s || exit 1; \
	echo "Build completed, binaries are in ./dist"

.PHONY: image
image: ## Builds and pushes container image (IMAGE_REGISTRY, IMAGE_TAG)
	@set -e; \
	echo "Building and pushing image to $(IMAGE_REGISTRY)/cns:$(IMAGE_TAG)"; \
	KO_DOCKER_REPO=$(IMAGE_REGISTRY) ko build --bare --sbom=none --tags=$(IMAGE_TAG) ./cmd/cnsctl

.PHONY: release
release: ## Runs the full release process with goreleaser
	@set -e; \
	goreleaser release --clean --config .goreleaser.yaml --fail-fast --timeout 10m0s

.PHONY: bump-major
bump-major: ## Bumps major version (1.2.3 → 2.0.0)
	tools/bump major

.PHONY: bump-minor
bump-minor: ## Bumps minor version (1.2.3 → 1.3.0)
	tools/bump minor

.PHONY: bump-patch
bump-patch: ## Bumps patch version (1.2.3 → 1.2.4)
	tools/bump patch

.PHONY: clean
clean: ## Cleans build artifacts (dist, coverage files)
	@rm -rf ./dist ./bin ./coverage.out
	@go clean ./...
	@echo "Cleaned build artifacts"

.PHONY: clean-all
clean-all: clean ## Deep cleans including Go module cache
	@echo "Cleaning module cache..."
	@go clean -modcache
	@echo "Deep clean completed"

.PHONY: cleanup
cleanup: ## Cleans up CNS Kubernetes resources (requires kubectl)
	tools/cleanup

.PHONY: demos
demos: ## Creates demo GIFs using VHS tool
	vhs docs/demos/videos/cli.tape -o docs/demos/videos/cli.gif

.PHONY: help
help: ## Displays available commands
	@echo "Available make targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
