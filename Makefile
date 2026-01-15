# Makefile for the cloud-native-stack project
# Purpose: Build, lint, test, and manage releases for the cloud-native-stack project.

REPO_NAME          := cloud-native-stack
VERSION            ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
IMAGE_REGISTRY     ?= ghcr.io/nvidia
IMAGE_TAG          ?= latest
YAML_FILES         := $(shell find . -type f \( -iname "*.yml" -o -iname "*.yaml" \) ! -path "./examples/*" ! -path "./~archive/*" ! -path "./bundles/*" ! -path "./.flox/*")
COMMIT             := $(shell git rev-parse HEAD)
BRANCH             := $(shell git rev-parse --abbrev-ref HEAD)
GO_VERSION	       := $(shell go env GOVERSION 2>/dev/null | sed 's/go//')
GOLINT_VERSION     = $(shell golangci-lint --version 2>/dev/null | awk '{print $$4}' | sed 's/golangci-lint version //' || echo "not installed")
KO_VERSION         = $(shell ko version 2>/dev/null || echo "not installed")
GORELEASER_VERSION = $(shell goreleaser --version 2>/dev/null | sed -n 's/^GitVersion:[[:space:]]*//p' || echo "not installed")


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

.PHONY: tidy
tidy: ## Updates Go modules all dependencies
	@set -e; \
	go fmt ./...; \
	go mod tidy

.PHONY: upgrade
upgrade: ## Upgrades all dependencies
	@set -e; \
	go get -u ./...; \
	go mod tidy

.PHONY: lint
lint: lint-go lint-yaml ## Lints the entire project
	@echo "Completed Go and YAML lints"

.PHONY: lint-go
lint-go: ## Lints the Go files
	@set -e; \
	echo "Running golangci-lint"; \
	golangci-lint -c .golangci.yaml run

.PHONY: lint-yaml
lint-yaml: ## Lints YAML files
	@if [ -n "$(YAML_FILES)" ]; then \
		yamllint -c .yamllint.yaml $(YAML_FILES); \
	else \
		echo "No YAML files found to lint."; \
	fi

.PHONY: test
test: ## Runs unit tests
	@set -e; \
	echo "Running tests with race detector"; \
	go test -count=1 -race -covermode=atomic -coverprofile=coverage.out ./... || exit 1; \
	echo "Test coverage"; \
	go tool cover -func=coverage.out

.PHONY: e2e
e2e: ## Runs integration tests
	@set -e; \
	echo "Running e2e integration tests"; \
	tools/e2e

.PHONY: scan
scan: ## Scans for source vulnerabilities
	@set -e; \
	echo "Doing static analysis"; \
	go vet ./...; \
	echo "Running vulnerability scan"; \
	grype dir:. --config .grype.yaml --fail-on high --quiet	

.PHONY: qualify
qualify: test lint e2e scan ## Qualifies the current codebase (test, lint, e2e, scan)
	@echo "Codebase qualification completed"

.PHONY: server
server: ## Starts a local development server
	@set -e; \
	echo "Starting local development server"; \
	LOG_LEVEL=debug go run cmd/cnsd/main.go

.PHONY: docs
docs: ## Generates and serves Go documentation on http://localhost:6060
	@set -e; \
	echo "Starting Go documentation server on http://localhost:6060"; \
	echo "Visit http://localhost:6060 to view docs"; \
	command -v pkgsite >/dev/null 2>&1 && pkgsite -http=:6060 || \
	(command -v godoc >/dev/null 2>&1 && godoc -http=:6060 || \
	(echo "Installing pkgsite..." && go install golang.org/x/pkgsite/cmd/pkgsite@latest && pkgsite -http=:6060))

.PHONY: build
build: tidy ## Builds the release for the current OS and architecture
	@set -e; \
	goreleaser build --clean --single-target --snapshot --timeout 10m0s || exit 1; \
	echo "Build completed, binaries are in ./dist"

.PHONY: image
image: ## Builds and pushes the container image (IMAGE_REGISTRY=ghcr.io/nvidia IMAGE_TAG=latest)
	@set -e; \
	echo "Building and pushing image to $(IMAGE_REGISTRY)/cns:$(IMAGE_TAG)"; \
	KO_DOCKER_REPO=$(IMAGE_REGISTRY) ko build --bare --sbom=none --tags=$(IMAGE_TAG) ./cmd/cnsctl

.PHONY: release
release: ## Runs the release process
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
clean: ## Cleans directories
	@set -e; \
	go clean -modcache; \
	go clean ./...; \
	rm -rf ./bin; \
	go get -u ./...; \
	go mod tidy; \
	echo "Cleaned directories"

.PHONY: help
help: ## Displays available commands
	@echo "Available make targets:"; \
	grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'