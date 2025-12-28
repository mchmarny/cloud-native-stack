# Contributing to NVIDIA Cloud Native Stack

Thank you for your interest in contributing to NVIDIA Cloud Native Stack! We welcome contributions from developers of all backgrounds, experience levels, and disciplines.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Project Architecture](#project-architecture)
- [Development Workflow](#development-workflow)
- [Building and Testing](#building-and-testing)
- [Code Quality Standards](#code-quality-standards)
- [Pull Request Process](#pull-request-process)
- [Developer Certificate of Origin](#developer-certificate-of-origin)

## Code of Conduct

This project follows NVIDIA's commitment to fostering an open and welcoming environment. Please be respectful and professional in all interactions.

## How Can I Contribute?

### Reporting Bugs

- Use the [GitHub issue tracker](https://github.com/NVIDIA/cloud-native-stack/issues) to report bugs
- Describe the issue clearly, including steps to reproduce
- Include relevant system information (OS, Go version, hardware)
- Attach logs or screenshots if applicable
- Check if the issue already exists before creating a new one

### Suggesting Enhancements

- Open an issue with the "enhancement" label
- Clearly describe the proposed feature and its use case
- Explain how it benefits the project and users
- Provide examples or mockups if applicable

### Improving Documentation

- Fix typos, clarify instructions, or add examples
- Update README.md for user-facing changes
- Update installation guides in [docs/install-guides](docs/install-guides)
- Enhance playbook documentation in [docs/playbooks](docs/playbooks)
- Update API documentation when endpoints change

### Contributing Code

- Fix bugs, add features, or improve performance
- Add new collectors for system configuration capture
- Enhance recipe generation logic
- Improve error handling and logging
- Follow the development workflow outlined below
- Ensure all tests pass and code meets quality standards

## Development Setup

### Prerequisites

- **Go**: Version 1.21 or higher ([download](https://golang.org/dl/))
- **golangci-lint**: Latest version ([installation](https://golangci-lint.run/usage/install/))
- **yamllint**: For YAML validation (`pip install yamllint`)
- **grype**: For vulnerability scanning ([installation](https://github.com/anchore/grype#installation))
- **goreleaser**: For building releases ([installation](https://goreleaser.com/install/))
- **make**: For build automation (usually pre-installed on Unix systems)
- **git**: For version control

### Clone the Repository

```bash
git clone https://github.com/NVIDIA/cloud-native-stack.git
cd cloud-native-stack
```

### Install Dependencies

```bash
# Download and update Go modules
make tidy

# Verify tool versions
make info
```

Example output:

```
version:        v0.7.6
commit:         abc123...
branch:         main
repo:           cloud-native-stack
go:             1.23.4
linter:         1.62.2
ko:             0.17.2
goreleaser:     2.5.1
```

## Project Architecture

### Directory Structure

```
cloud-native-stack/
├── cmd/                      # Entry points
│   ├── eidos/               # CLI binary
│   └── eidos-api-server/    # API server binary
├── pkg/                      # Core packages
│   ├── api/                 # HTTP API layer and handlers
│   ├── bundler/             # Bundle generation framework
│   │   ├── examples/       # Example bundler implementations
│   │   └── gpuoperator/    # GPU Operator bundler with templates
│   ├── cli/                 # CLI commands (urfave/cli v3)
│   ├── collector/           # System configuration collectors
│   │   ├── gpu/            # GPU hardware collectors
│   │   ├── k8s/            # Kubernetes API collectors
│   │   ├── os/             # Operating system collectors
│   │   └── systemd/        # SystemD service collectors
│   ├── logging/             # Structured logging (slog)
│   ├── measurement/         # Measurement types and utilities
│   ├── recipe/              # Recipe generation logic
│   │   ├── header/         # Common header types
│   │   └── version/        # Semantic version parsing
│   ├── serializer/          # Output formatting (JSON/YAML/table)
│   ├── server/              # HTTP server implementation
│   └── snapshotter/         # Snapshot orchestration
├── api/                      # API specifications
│   └── eidos/               # OpenAPI/Swagger definitions
├── deployments/             # Kubernetes manifests
│   └── eidos-agent/         # Agent Job and RBAC
├── docs/                    # Documentation
│   ├── install-guides/      # Platform-specific guides
│   ├── playbooks/           # Ansible automation
│   ├── optimizations/       # Performance tuning
│   └── troubleshooting/     # Common issues
├── examples/                # Example configurations and comparisons
├── infra/                   # Infrastructure as code (Terraform)
├── tools/                   # Build and release scripts
├── .goreleaser.yaml         # Release configuration
├── .golangci.yaml           # Linter configuration
└── Makefile                 # Build automation

```

### Key Components

#### CLI (`eidos`)
- **Location**: `cmd/eidos/main.go` → `pkg/cli/`
- **Framework**: [urfave/cli v3](https://github.com/urfave/cli)
- **Commands**: `snapshot`, `recipe`
- **Purpose**: User-facing tool for system snapshots and recipe generation (supports both query and snapshot modes)
- **Output**: Supports JSON, YAML, and table formats

#### API Server
- **Location**: `cmd/eidos-api-server/main.go` → `pkg/server/`, `pkg/api/`
- **Endpoints**: 
  - `GET /v1/recipe` - Generate configuration recipes
  - `GET /health` - Liveness probe
  - `GET /ready` - Readiness probe
  - `GET /metrics` - Prometheus metrics
- **Purpose**: HTTP service for recipe generation with rate limiting and observability
- **Deployment**: https://cns.dgxc.io

#### Collectors
- **Location**: `pkg/collector/`
- **Pattern**: Factory-based with dependency injection
- **Types**: 
  - **SystemD**: Service states (containerd, docker, kubelet)
  - **OS**: 4 subtypes - grub, sysctl, kmod, release
  - **Kubernetes**: Node info, server version, images, ClusterPolicy
  - **GPU**: Hardware info, driver version, MIG settings
- **Purpose**: Parallel collection of system configuration data
- **Context Support**: All collectors respect context cancellation

#### Recipe Engine
- **Location**: `pkg/recipe/`
- **Purpose**: Generate optimized configurations using base-plus-overlay model
- **Modes**:
  - **Query Mode**: Direct recipe generation from system parameters
  - **Snapshot Mode**: Extract query from snapshot → Build recipe → Return recommendations
- **Input**: OS, OS version, kernel, K8s service/version, GPU type, workload intent
- **Output**: Recipe with matched rules and configuration measurements
- **Data Source**: Embedded YAML configuration (`recipe/data/data-v1.yaml`)
- **Query Extraction**: Parses K8s, OS, GPU measurements from snapshots to construct recipe queries

#### Snapshotter
- **Location**: `pkg/snapshotter/`
- **Purpose**: Orchestrate parallel collection of system measurements
- **Output**: Complete snapshot with metadata and all collector measurements
- **Usage**: CLI command, Kubernetes Job agent
- **Format**: Structured snapshot (snapshot.dgxc.io/v1)

#### Bundler Framework
- **Location**: `pkg/bundler/`
- **Pattern**: Registry-based with pluggable bundler implementations
- **API**: Object-oriented with functional options (DefaultBundler.New())
- **Purpose**: Generate deployment bundles from recipes (Helm values, K8s manifests, scripts)
- **Bundlers**:
  - **GPU Operator**: Generates complete GPU Operator deployment bundle
    - Helm values.yaml with version management
    - Kubernetes ClusterPolicy manifest
    - Installation/uninstallation scripts
    - README with deployment instructions
    - SHA256 checksums for verification
- **Features**:
  - Template-based generation with go:embed
  - Functional options pattern for configuration (WithBundlerTypes, WithFailFast, WithConfig, WithRegistry)
  - **Parallel execution** (all bundlers run concurrently)
  - Empty bundlerTypes = all registered bundlers (dynamic discovery)
  - Fail-fast or error collection modes
  - Prometheus metrics for observability
  - Context-aware execution with cancellation support
- **Extensibility**: Implement `Bundler` interface and self-register in init() to add new bundle types

### Common Make Targets

```bash
# Development
make tidy         # Format code and update dependencies
make build        # Build binaries for current platform
make server       # Start API server locally (debug mode)

# Testing
make test         # Run unit tests with coverage
make test-race    # Run tests with race detection
make qualify      # Run tests, lints, and scans (full check)

# Code Quality
make lint         # Lint Go and YAML files
make lint-go      # Lint Go files only
make lint-yaml    # Lint YAML files only
make scan         # Security and vulnerability scanning

# Dependency Management
make upgrade      # Upgrade all dependencies
make info         # Show project and tool versions

# Releases (CI only)
make release      # Build multi-platform release artifacts
make snapshot     # Create release snapshot
make bump-major   # Bump major version
make bump-minor   # Bump minor version
make bump-patch   # Bump patch version

# Utilities
make help         # Show all available targets
```

## Development Workflow

### 1. Create a Branch

Use descriptive branch names:

```bash
# For new features
git checkout -b feature/add-gpu-collector

# For bug fixes
git checkout -b fix/snapshot-crash-on-empty-gpu

# For documentation
git checkout -b docs/update-contributing-guide
```

### 2. Make Changes

Follow these principles:
- **Small, focused commits**: Each commit should address one logical change
- **Clear commit messages**: Use imperative mood (e.g., "Add GPU collector" not "Added GPU collector")
- **Test as you go**: Write tests alongside your code
- **Document your code**: Add comments for exported functions and complex logic

### 3. Add a New Collector (Example)

If adding a new system collector (like the OS release collector added in v0.7.0):

1. Create the collector in `pkg/collector/os/`:
```go
// pkg/collector/os/release.go
package os

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"
)

// collectRelease reads and parses /etc/os-release
func (c *Collector) collectRelease(ctx context.Context) (*measurement.Subtype, error) {
    data := make(map[string]measurement.Reading)
    
    file, err := os.Open("/etc/os-release")
    if err != nil {
        return nil, fmt.Errorf("failed to open /etc/os-release: %w", err)
    }
    defer file.Close()
    
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        
        key := parts[0]
        value := strings.Trim(parts[1], `"`)
        data[key] = measurement.Reading{Value: value}
    }
    
    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("error reading /etc/os-release: %w", err)
    }
    
    return &measurement.Subtype{
        Name: "release",
        Data: data,
    }, nil
}
```

2. Update the main collector to include the new subtype:
```go
// pkg/collector/os/os.go
func (c *Collector) Collect(ctx context.Context) ([]*measurement.Measurement, error) {
    // Collect all OS subtypes in parallel
    grubSubtype, _ := c.collectGrub(ctx)
    sysctlSubtype, _ := c.collectSysctl(ctx)
    kmodSubtype, _ := c.collectKmod(ctx)
    releaseSubtype, _ := c.collectRelease(ctx) // New subtype
    
    return []*measurement.Measurement{{
        Type: measurement.TypeOS,
        Subtypes: []*measurement.Subtype{
            grubSubtype,
            sysctlSubtype,
            kmodSubtype,
            releaseSubtype, // Add to list
        },
    }}, nil
}
```

3. Add tests for the new collector:
```go
// pkg/collector/os/release_test.go
func TestCollectRelease(t *testing.T) {
    c := NewCollector()
    ctx := context.Background()
    
    subtype, err := c.collectRelease(ctx)
    if err != nil {
        t.Fatalf("collectRelease() error = %v", err)
    }
    
    // Verify expected fields exist
    expectedFields := []string{"ID", "VERSION_ID", "PRETTY_NAME"}
    for _, field := range expectedFields {
        if _, exists := subtype.Data[field]; !exists {
            t.Errorf("expected field %q not found", field)
        }
    }
    
    // Verify subtype name
    if subtype.Name != "release" {
        t.Errorf("expected subtype name 'release', got %q", subtype.Name)
    }
}
```

4. Update integration tests to expect 4 OS subtypes instead of 3:
```go
// pkg/collector/os/os_test.go
func TestOSCollector(t *testing.T) {
    measurements, err := c.Collect(ctx)
    if err != nil {
        t.Fatalf("Collect() error = %v", err)
    }
    
    // Should return 4 subtypes: grub, sysctl, kmod, release
    if len(measurements[0].Subtypes) != 4 {
        t.Errorf("expected 4 subtypes, got %d", len(measurements[0].Subtypes))
    }
}
```

### Example: Version Parser with Vendor Extras

When adding version parsing support for vendor-specific formats:

```go
// pkg/recipe/version/version.go
type Version struct {
    Major  int
    Minor  int
    Patch  int
    Extras string // New field for vendor suffixes
}

func ParseVersion(s string) (*Version, error) {
    // Remove 'v' prefix if present
    s = strings.TrimPrefix(s, "v")
    
    // Find position of extras (after digits, before first dash or plus)
    extrasPos := -1
    for i, c := range s {
        if (c == '-' || c == '+') && i > 0 && isDigit(rune(s[i-1])) {
            extrasPos = i
            break
        }
    }
    
    // Split version from extras
    versionPart := s
    extras := ""
    if extrasPos != -1 {
        versionPart = s[:extrasPos]
        extras = s[extrasPos:]
    }
    
    // Parse Major.Minor.Patch
    parts := strings.Split(versionPart, ".")
    // ... parse logic ...
    
    return &Version{
        Major:  major,
        Minor:  minor,
        Patch:  patch,
        Extras: extras,
    }, nil
}

// String returns the version without extras for clean comparison
func (v *Version) String() string {
    return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
```

Tests for version parsing with extras:
```go
func TestParseVersionWithExtras(t *testing.T) {
    tests := []struct {
        input        string
        wantMajor    int
        wantMinor    int
        wantPatch    int
        wantExtras   string
    }{
        {"6.8.0-1028-aws", 6, 8, 0, "-1028-aws"},
        {"v1.33.5-eks-3025e55", 1, 33, 5, "-eks-3025e55"},
        {"v1.28.0-gke.1337000", 1, 28, 0, "-gke.1337000"},
        {"1.29.2-hotfix.20240322", 1, 29, 2, "-hotfix.20240322"},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            v, err := ParseVersion(tt.input)
            if err != nil {
                t.Fatalf("ParseVersion(%q) error = %v", tt.input, err)
            }
            if v.Major != tt.wantMajor || v.Minor != tt.wantMinor || 
               v.Patch != tt.wantPatch || v.Extras != tt.wantExtras {
                t.Errorf("got %d.%d.%d%s, want %d.%d.%d%s",
                    v.Major, v.Minor, v.Patch, v.Extras,
                    tt.wantMajor, tt.wantMinor, tt.wantPatch, tt.wantExtras)
            }
        })
    }
}
```

### 4. Run Tests

```bash
# Run all tests
make test

# Run tests for a specific package
go test ./pkg/collector/... -v

# Run tests with race detection
go test -race ./...

# Check test coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 5. Lint Your Code

```bash
# Run all linters
make lint

# Fix auto-fixable issues
golangci-lint run --fix

# Check specific files
golangci-lint run pkg/collector/network.go
```

Common linting issues and fixes:
- **Unused variables/imports**: Remove them
- **Error handling**: Never ignore errors without explicit comment
- **Naming conventions**: Use camelCase for unexported, PascalCase for exported
- **Line length**: Keep lines under 120 characters

### 6. Test Locally

```bash
# Build for current platform
make build

# Test CLI commands
./dist/eidos_*/eidos snapshot
./dist/eidos_*/eidos recipe --os ubuntu --service eks

# Start API server
make server

# Test API endpoints (in another terminal)
curl http://localhost:8080/healthz
curl "http://localhost:8080/v1/recipe?os=ubuntu&service=eks"
```

### 7. Run Security Scans

```bash
# Run vulnerability scan
make scan

# Manual vulnerability check
grype dir:. --fail-on high
```

### 8. Full Qualification

Before submitting a PR:

```bash
# Run everything
make qualify

# This runs:
# - make test   (unit tests with coverage)
# - make lint   (Go and YAML linting)
# - make scan   (vulnerability scanning)
```

All checks must pass before PR submission.

## Building and Testing

### Local Development

```bash
# Quick build for local testing
make build

# Build outputs to dist/
ls -lh dist/

# Example output:
# dist/
#   eidos_darwin_arm64/
#     eidos
#   eidos-api-server_darwin_arm64/
#     eidos-api-server
```

### Running the CLI

```bash
# Help
./dist/eidos_*/eidos --help

# Snapshot (outputs to stdout)
eidos snapshot --format yaml
eidos snapshot --output system.yaml --format json

# Recipe generation
eidos recipe --os ubuntu --service eks --gpu h100
eidos recipe \
  --os ubuntu \
  --osv 24.04 \
  --kernel 6.8 \
  --service eks \
  --k8s 1.33 \
  --gpu h100 \
  --intent training \
  --context \
  --format yaml

# Generate recipe from snapshot
eidos recipe --snapshot system.yaml --intent training
eidos recipe -f system.yaml -i inference -o recipe.yaml
```

### Running the API Server

```bash
# Development mode with debug logging
make server

# Custom configuration
PORT=8080 LOG_LEVEL=debug go run cmd/eidos-api-server/main.go

# Test endpoints
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8080/metrics
curl "http://localhost:8080/v1/recipe?os=ubuntu&gpu=h100&intent=training"
curl "http://localhost:8080/v1/recipe?os=ubuntu&osv=24.04&service=eks&gpu=gb200&context=true"

# Test with jq
curl -s "http://localhost:8080/v1/recipe?gpu=h100" | jq '.matchedRules'
```

### Testing in Kubernetes

Build and deploy the agent locally:

```bash
# Build container image with ko
ko build --local ./cmd/eidos-api-server

# Or build with Docker
docker build -t eidos:dev -f Dockerfile .

# Deploy agent
kubectl apply -f deployments/eidos-agent/1-deps.yaml
kubectl apply -f deployments/eidos-agent/2-job.yaml

# Update job image for testing
kubectl set image job/eidos -n gpu-operator eidos=<local-image>

# Check logs
kubectl get jobs -n gpu-operator
kubectl logs -n gpu-operator job/eidos -f

# Get snapshot output
kubectl logs -n gpu-operator job/eidos > snapshot.yaml
```

### Adding a New Bundler

If adding a new deployment bundler (like a Network Operator bundler):

1. Create bundler package in `pkg/bundler/<bundler-name>/`:
```go
// pkg/bundler/networkoperator/bundler.go
package networkoperator

import (
    "context"
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler"
    "github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const bundlerType = bundler.BundleType("network-operator")

func init() {
    // Self-register bundler instance in global registry
    bundler.Register(bundlerType, &Bundler{})
}

type Bundler struct {
    // Bundler should be stateless or use synchronization for thread-safety
}

func NewBundler() *Bundler {
    return &Bundler{}
}

func (b *Bundler) Type() bundler.BundleType {
    return bundlerType
}

func (b *Bundler) Validate(ctx context.Context, r *recipe.Recipe) error {
    // Validate recipe has required measurements
    return nil
}

func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, 
    outputDir string) (*bundler.BundleResult, error) {
    
    // Extract data from recipe measurements
    config := b.extractConfig(r)
    
    // Generate bundle files in parallel
    // Return BundleResult with files, checksums, metadata
    return result, nil
}
```

2. Create templates directory with embedded templates:
```
pkg/bundler/networkoperator/templates/
├── values.yaml.tmpl      # Helm chart values
├── manifest.yaml.tmpl    # Kubernetes manifests
├── install.sh.tmpl       # Installation script
└── README.md.tmpl        # Documentation
```

3. Embed templates using go:embed:
```go
//go:embed templates/*.tmpl
var templatesFS embed.FS

func (b *Bundler) renderTemplate(name string, 
    data map[string]interface{}) (string, error) {
    
    tmpl, err := template.New(name).
        ParseFS(templatesFS, "templates/"+name+".tmpl")
    if err != nil {
        return "", fmt.Errorf("failed to parse template: %w", err)
    }
    
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to execute template: %w", err)
    }
    
    return buf.String(), nil
}
```

4. Write comprehensive tests:
```go
// pkg/bundler/networkoperator/bundler_test.go
func TestBundler_Make(t *testing.T) {
    tests := []struct {
        name    string
        recipe  *recipe.Recipe
        wantErr bool
    }{
        {
            name:    "valid recipe",
            recipe:  createTestRecipe(),
            wantErr: false,
        },
        {
            name:    "missing required measurements",
            recipe:  &recipe.Recipe{},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            tmpDir := t.TempDir()
            
            b := NewBundler()
            result, err := b.Make(ctx, tt.recipe, tmpDir)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Make() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !tt.wantErr {
                // Verify expected files exist
                if len(result.Files) == 0 {
                    t.Error("expected generated files, got none")
                }
                
                // Verify checksums
                if len(result.Checksums) != len(result.Files) {
                    t.Error("checksum count mismatch")
                }
            }
        })
    }
}

func createTestRecipe() *recipe.Recipe {
    return &recipe.Recipe{
        APIVersion: "v1",
        Kind:       "Recipe",
        Measurements: []*measurement.Measurement{
            {
                Type: measurement.TypeK8s,
                Subtypes: []*measurement.Subtype{
                    {
                        Name: "image",
                        Data: map[string]measurement.Reading{
                            "network-operator": measurement.Str("v25.4.0"),
                            "ofed-driver":      measurement.Str("24.07"),
                        },
                    },
                    {
                        Name: "config",
                        Data: map[string]measurement.Reading{
                            "rdma":   measurement.Bool(true),
                            "sr-iov": measurement.Bool(true),
                        },
                    },
                },
            },
        },
    }
}
```

5. Test bundle generation:
```bash
# Build CLI with new bundler
make build

# Test bundle generation
./dist/eidos_*/eidos bundle \
  --recipe examples/recipe.yaml \
  --bundler network-operator \
  --output ./test-bundles \
  --dry-run

# Verify bundle structure
ls -la test-bundles/network-operator/
```

**Bundler Best Practices**:
- Use functional options pattern for configuration
- Bundlers execute in parallel automatically
- Embed templates with go:embed for portability
- Compute SHA256 checksums for verification
- Check context cancellation for long operations
- Add structured logging with slog
- Expose Prometheus metrics
- Write integration tests with real recipes
- Keep bundlers stateless or use synchronization for thread-safety

## Code Quality Standards

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting (automated via `make tidy`)
- Write clear, self-documenting code with meaningful names
- Keep functions small and focused (single responsibility)
- Add godoc comments for all exported types, functions, and methods

Example:
```go
// Collector defines the interface for system configuration collectors.
// Each collector is responsible for gathering specific system information
// and returning it in a structured format.
type Collector interface {
    // Name returns the unique identifier for this collector.
    Name() string
    
    // Collect gathers configuration data and returns it.
    // Returns an error if collection fails.
    Collect(ctx context.Context) (interface{}, error)
}
```

### Testing Requirements

- **Coverage**: Aim for meaningful test coverage (current: ~60%, target: >70%)
- **Unit tests**: Test all public functions and methods
- **Table-driven tests**: Use for multiple test cases
- **Integration tests**: Test collector interactions with real/fake clients
- **Error cases**: Test error conditions and edge cases
- **Context handling**: Test context cancellation and timeouts
- **Mocks**: Use fake clients for external dependencies (Kubernetes client-go fakes)

Example test structure:
```go
func TestCollectorName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "test", false},
        {"empty input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Function() error = %v, wantErr %v", err, tt.wantErr)
            }
            if result != tt.expected {
                t.Errorf("Function() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Error Handling

- **Always check errors**: Never ignore errors without explicit `// nolint:errcheck` comment
- **Wrap errors**: Add context using `fmt.Errorf` with `%w`
- **Sentinel errors**: Define package-level errors for common cases
- **Error checking**: Use `errors.Is()` and `errors.As()` for wrapped errors

Example:
```go
var ErrInvalidConfig = errors.New("invalid configuration")

func Process(config Config) error {
    if config.Name == "" {
        return fmt.Errorf("%w: name is required", ErrInvalidConfig)
    }
    
    if err := validate(config); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    return nil
}
```

### Logging

- **Use structured logging**: Use `pkg/logging` package with slog
- **Appropriate levels**:
  - `Debug`: Detailed diagnostic information
  - `Info`: General informational messages
  - `Warn`: Warning messages for recoverable issues
  - `Error`: Error messages for failures
- **Context**: Include relevant context (IDs, names, paths)
- **Security**: Never log sensitive information (passwords, keys, tokens)

Example:
```go
import "log/slog"

func ProcessRequest(ctx context.Context, id string) error {
    slog.Debug("processing request", "id", id)
    
    if err := doWork(ctx, id); err != nil {
        slog.Error("failed to process request", "id", id, "error", err)
        return err
    }
    
    slog.Info("request processed successfully", "id", id)
    return nil
}
```

### Context Propagation

- Pass `context.Context` as the first parameter to functions performing I/O
- Respect context cancellation in long-running operations
- Use `context.WithTimeout` or `context.WithDeadline` for time-bound operations
- Avoid storing context in structs (except for special cases)

### Dependencies

- **Minimize**: Use standard library when possible
- **Vet carefully**: Review licenses and maintenance status
- **Keep updated**: Regularly update dependencies (`make upgrade`)
- **Document**: Explain why external dependencies are needed in PR description

Current key dependencies:
- `github.com/urfave/cli/v3` - CLI framework
- `k8s.io/client-go` - Kubernetes API client
- `k8s.io/api` - Kubernetes API types
- `golang.org/x/sync/errgroup` - Concurrent error handling
- `golang.org/x/time/rate` - Rate limiting
- `gopkg.in/yaml.v3` - YAML parsing and generation
- `github.com/stretchr/testify` - Testing assertions
- Standard library for most core functionality

## Pull Request Process

### Before Submitting

**1. Ensure all checks pass:**
```bash
make qualify
```

**2. Update documentation:**
- [ ] README.md for user-facing changes
- [ ] CONTRIBUTING.md for developer workflow changes
- [ ] Code comments and godoc
- [ ] docs/ for guides or playbooks

**3. Commit with DCO sign-off:**
```bash
git add .
git commit -s -m "Add network collector for system configuration

- Implement NetworkCollector interface
- Add unit tests with 80% coverage
- Update factory registration
- Document collector usage

Fixes #123"
```

**Important**: Always use `-s` flag for DCO sign-off.

**4. Push to your fork:**
```bash
git push origin feature/your-feature-name
```

### Creating the Pull Request

1. Navigate to [NVIDIA/cloud-native-stack](https://github.com/NVIDIA/cloud-native-stack)
2. Click "New Pull Request"
3. Select your branch
4. Fill out the PR template:

**Title**: Clear, concise (e.g., "Add network collector" or "Fix GPU detection crash")

**Description**:
```markdown
## Summary
Brief description of changes

## Changes
- Bullet list of specific changes
- What was added/modified/removed

## Related Issues
Fixes #123

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests performed
- [ ] Manual testing on Ubuntu 24.04
- [ ] API endpoints tested

## Breaking Changes
None / Describe any breaking changes

## Checklist
- [x] All tests pass (`make test`)
- [x] Linting passes (`make lint`)
- [x] Security scan passes (`make scan`)
- [x] Documentation updated
- [x] Commits are signed off (DCO)
```

### Review Process

1. **Automated checks**: CI will run tests, lints, and scans
2. **Maintainer review**: A maintainer will review your code
3. **Feedback**: Address any requested changes by pushing new commits
4. **Approval**: Once approved, a maintainer will merge your PR
5. **Celebration**: Your contribution is now part of the project! 🎉

### Addressing Feedback

```bash
# Make requested changes
vim pkg/collector/network.go

# Test changes
make test

# Commit with DCO
git commit -s -m "Address review feedback: improve error handling"

# Push to update PR
git push origin feature/your-feature-name
```

### After Merging

```bash
# Update your local repository
git checkout main
git pull upstream main

# Delete your feature branch
git branch -d feature/your-feature-name
git push origin --delete feature/your-feature-name
```

## Developer Certificate of Origin

All contributions must include a DCO sign-off to certify that you have the right to submit the contribution under the project's license.

### How to Sign Off

Add the `-s` flag to your commit:

```bash
git commit -s -m "Your commit message"
```

This adds a "Signed-off-by" line:
```
Signed-off-by: Jane Developer <jane@example.com>
```

The sign-off certifies that you agree to the DCO below.

### Developer Certificate of Origin 1.1

```
Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

### Amending Commits

If you forget to sign off, amend your commit:

```bash
git commit --amend --signoff
git push --force-with-lease origin feature/your-branch
```

## Tips for Contributors

### First-Time Contributors

- Start with "good first issue" labeled issues
- Read through existing code to understand patterns
- Don't hesitate to ask questions in issues or PRs
- Test thoroughly before submitting

### Writing Good Commit Messages

```
Short summary (50 chars or less)

More detailed explanation if needed. Wrap at 72 characters.
Explain the problem being solved and why this approach was chosen.

- Bullet points are fine
- Use present tense ("Add feature" not "Added feature")
- Reference issues: "Fixes #123" or "Related to #456"

Signed-off-by: Your Name <your@email.com>
```

### Debugging Tips

```bash
# Enable debug logging
./dist/eidos_*/eidos --debug snapshot

# Run specific test with verbose output
go test -v ./pkg/collector/ -run TestGPUCollector

# Print test coverage by function
go test -coverprofile=coverage.out ./pkg/collector/
go tool cover -func=coverage.out

# Profile CPU usage
go test -cpuprofile=cpu.prof ./pkg/collector/
go tool pprof cpu.prof
```

## Additional Resources

### Project Documentation
- [README.md](README.md) - User documentation and quick start
- [docs/README.md](docs/README.md) - Comprehensive platform documentation
- [docs/install-guides](docs/install-guides) - Platform-specific installation
- [docs/playbooks](docs/playbooks) - Ansible automation guides
- [docs/optimizations](docs/optimizations) - Performance tuning guides
- [docs/troubleshooting](docs/troubleshooting) - Common issues and solutions

### External Resources
- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [urfave/cli Documentation](https://cli.urfave.org/)

### Getting Help

- **GitHub Issues**: [Create an issue](https://github.com/NVIDIA/cloud-native-stack/issues/new)
- **Discussions**: Check existing discussions and open new ones
- **Email**: For security issues, contact the maintainers privately

## Questions?

If you have questions about contributing:
- Open a GitHub issue with the "question" label
- Check existing issues for similar questions
- Review this guide and project documentation
- Look at recent merged PRs for examples

Thank you for contributing to NVIDIA Cloud Native Stack! Your efforts help improve GPU-accelerated infrastructure for everyone.

