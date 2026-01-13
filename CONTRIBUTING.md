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
- Update installation guides in [~archive/cns-v1/install-guides](~archive/cns-v1/install-guides)
- Enhance playbook documentation in [~archive/cns-v1/playbooks](~archive/cns-v1/playbooks)
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
│   ├── cnsctl/               # CLI binary
│   └── cnsd/    # API server binary
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
│   └── cns/               # OpenAPI/Swagger definitions
├── deployments/             # Kubernetes manifests
│   └── cns-agent/         # Agent Job and RBAC
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

#### CLI (`cnsctl`)
- **Location**: `cmd/cnsctl/main.go` → `pkg/cli/`
- **Framework**: [urfave/cli v3](https://github.com/urfave/cli)
- **Commands**: `snapshot`, `recipe`
- **Purpose**: User-facing tool for system snapshots and recipe generation (supports both query and snapshot modes)
- **Output**: Supports JSON, YAML, and table formats

#### API Server
- **Location**: `cmd/cnsd/main.go` → `pkg/server/`, `pkg/api/`
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
- **Format**: Structured snapshot (cns.nvidia.com/v1alpha1)

#### Bundler Framework
- **Location**: `pkg/bundler/`
- **Pattern**: Registry-based with pluggable bundler implementations
- **API**: Object-oriented with functional options (DefaultBundler.New())
- **Purpose**: Generate deployment bundles from recipes (Helm values, K8s manifests, scripts)
- **Available Bundlers**:
  - **GPU Operator**: Generates complete GPU Operator deployment bundle
    - Helm values.yaml with version management
    - Kubernetes ClusterPolicy manifest
    - Installation/uninstallation scripts
    - README with deployment instructions
    - SHA256 checksums for verification
  - **Network Operator**: Generates Network Operator deployment bundle
    - Helm values.yaml for RDMA and SR-IOV configuration
    - NICClusterPolicy manifest
  - **Cert-Manager**: Generates cert-manager deployment bundle
    - Helm values.yaml with resource configuration
  - **NVSentinel**: Generates NVSentinel deployment bundle
    - Helm values.yaml
  - **Skyhook**: Generates Skyhook node optimization bundle
    - Helm values.yaml
    - Skyhook CR manifest
- **Features**:
  - Template-based generation with go:embed
  - Functional options pattern for configuration (WithBundlerTypes, WithFailFast, WithConfig, WithRegistry)
  - **Parallel execution** (all bundlers run concurrently)
  - Empty bundlerTypes = all registered bundlers (dynamic discovery)
  - Fail-fast or error collection modes
  - Prometheus metrics for observability
  - Context-aware execution with cancellation support
  - **Value overrides**: CLI `--set bundler:path.to.field=value` allows runtime customization
  - **Node scheduling**: `--system-node-selector`, `--accelerated-node-selector`, and toleration flags for workload placement
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
./dist/cns_*/cnsctl snapshot
./dist/cns_*/cnsctl recipe --os ubuntu --service eks

# Test snapshot with ConfigMap output (requires cluster access)
./dist/cns_*/cnsctl snapshot --output cm://gpu-operator/test-snapshot

# Test agent deployment mode
./dist/cns_*/cnsctl snapshot --deploy-agent \
  --namespace gpu-operator \
  --node-selector nvidia.com/gpu.present=true

# Test recipe with ConfigMap input
./dist/cns_*/cnsctl recipe \
  --snapshot cm://gpu-operator/test-snapshot \
  --intent training

# Test recipe with custom kubeconfig
./dist/cns_*/cnsctl recipe \
  --snapshot cm://gpu-operator/test-snapshot \
  --kubeconfig ~/.kube/dev-cluster \
  --intent training

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
#   cns_darwin_arm64/
#     cnsctl
#   cnsd_darwin_arm64/
#     cnsd
```

### Running the CLI

```bash
# Help
./dist/cns_*/cnsctl --help

# STEP 1: Snapshot - Capture system configuration
cnsctl snapshot --format yaml
cnsctl snapshot --output system.yaml --format json
cnsctl snapshot --output cm://gpu-operator/cns-snapshot --format yaml  # ConfigMap output

# Agent deployment mode (Kubernetes Job on cluster nodes)
cnsctl snapshot --deploy-agent
cnsctl snapshot --deploy-agent --output cm://gpu-operator/cns-snapshot
cnsctl snapshot --deploy-agent --kubeconfig ~/.kube/prod-cluster

# Agent deployment with node targeting
# Note: All taints are tolerated by default, only specify --toleration to restrict
cnsctl snapshot --deploy-agent \
  --namespace gpu-operator \
  --node-selector accelerator=nvidia-h100 \
  --toleration nvidia.com/gpu:NoSchedule \
  --timeout 10m

# STEP 2: Recipe - Generate optimized configuration
# Query mode: Direct generation from parameters
cnsctl recipe --os ubuntu --service eks --gpu h100
cnsctl recipe \
  --os ubuntu \
  --osv 24.04 \
  --kernel 6.8 \
  --service eks \
  --k8s 1.33 \
  --gpu h100 \
  --intent training \
  --context \
  --format yaml

# Snapshot mode: Generate recipe from captured snapshot
cnsctl recipe --snapshot system.yaml --intent training
cnsctl recipe -s system.yaml -i inference -o recipe.yaml
cnsctl recipe -s cm://gpu-operator/cns-snapshot -i training -o cm://gpu-operator/cns-recipe  # ConfigMap I/O

# With custom kubeconfig for ConfigMap access
cnsctl recipe \
  -s cm://gpu-operator/cns-snapshot \
  --kubeconfig ~/.kube/prod-cluster \
  -i training \
  -o recipe.yaml

# STEP 3: Bundle - Create deployment artifacts
cnsctl bundle --recipe recipe.yaml --output ./bundles
cnsctl bundle -r recipe.yaml -b gpu-operator -o ./deployment
cnsctl bundle -r cm://gpu-operator/cns-recipe -o ./bundles  # ConfigMap input

# Override bundle values at generation time
cnsctl bundle -r recipe.yaml -b gpu-operator \
  --set gpuoperator:gds.enabled=true \
  --set gpuoperator:driver.version=570.86.16 \
  -o ./bundles

# Multiple bundlers with overrides
cnsctl bundle -r recipe.yaml \
  -b gpu-operator \
  -b network-operator \
  --set gpuoperator:mig.strategy=mixed \
  --set networkoperator:rdma.enabled=true \
  -o ./bundles
```

### Complete End-to-End Workflow

Here's a complete example showing all four steps:

```bash
# 1. Capture system configuration
cnsctl snapshot --output snapshot.yaml

echo "Snapshot captured:"
ls -lh snapshot.yaml

# 2. Generate optimized recipe for training workloads
cnsctl recipe \
  --snapshot snapshot.yaml \
  --intent training \
  --format yaml \
  --output recipe.yaml

echo "Recipe generated:"
cat recipe.yaml | grep "matchedRules" -A 5

# 3. Validate recipe constraints against snapshot
cnsctl validate \
  --recipe recipe.yaml \
  --snapshot snapshot.yaml

echo "Validation complete"

# 4. Create deployment bundle
cnsctl bundle \
  --recipe recipe.yaml \
  --bundlers gpu-operator \
  --output ./bundles

echo "Bundle generated:"
tree bundles/

# 5. Deploy to cluster
cd bundles/gpu-operator
cat README.md  # Review deployment instructions
sha256sum -c checksums.txt  # Verify file integrity
chmod +x scripts/install.sh
./scripts/install.sh  # Deploy GPU Operator

# 5. Monitor deployment
kubectl get pods -n gpu-operator
kubectl logs -n gpu-operator -l app=nvidia-operator-validator
```

**Alternative: ConfigMap-based Workflow (for Kubernetes Jobs)**

When running in Kubernetes, you can use ConfigMap URIs to avoid file dependencies:

```bash
# 1. Capture snapshot directly to ConfigMap (agent deployment mode)
cnsctl snapshot --deploy-agent -o cm://gpu-operator/cns-snapshot

# Alternative: Manual kubectl deployment
cnsctl snapshot -o cm://gpu-operator/cns-snapshot

# 2. Generate recipe from ConfigMap snapshot to ConfigMap output
cnsctl recipe -s cm://gpu-operator/cns-snapshot --intent training -o cm://gpu-operator/cns-recipe

# With custom kubeconfig
cnsctl recipe \
  -s cm://gpu-operator/cns-snapshot \
  --kubeconfig ~/.kube/config \
  --intent training \
  -o cm://gpu-operator/cns-recipe

# 3. Create bundle from ConfigMap recipe
cnsctl bundle -r cm://gpu-operator/cns-recipe -b gpu-operator -o ./bundles

# 4. Verify ConfigMap data
kubectl get configmap cns-snapshot -n gpu-operator -o yaml
kubectl get configmap cns-recipe -n gpu-operator -o yaml
```

**Expected Bundle Structure:**
```
bundles/gpu-operator/
├── values.yaml              # Helm chart configuration
├── manifests/
│   └── clusterpolicy.yaml  # ClusterPolicy custom resource
├── scripts/
│   ├── install.sh          # Automated installation
│   └── uninstall.sh        # Cleanup script
├── README.md                # Deployment guide
└── checksums.txt            # SHA256 checksums
```

### Running the API Server

```bash
# Development mode with debug logging
make server

# Custom configuration
PORT=8080 LOG_LEVEL=debug go run cmd/cnsd/main.go

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
ko build --local ./cmd/cnsd

# Or build with Docker
docker build -t cns:dev -f Dockerfile .

# Deploy agent
kubectl apply -f deployments/cns-agent/1-deps.yaml
kubectl apply -f deployments/cns-agent/2-job.yaml

# Update job image for testing
kubectl set image job/cns -n gpu-operator cns=<local-image>

# Check status and logs
kubectl get jobs -n gpu-operator
kubectl logs -n gpu-operator job/cns

# Get snapshot from ConfigMap
kubectl get configmap cns-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > snapshot.yaml

# Verify ConfigMap was created
kubectl get configmap cns-snapshot -n gpu-operator -o yaml
```

### End-to-End Testing

The `tools/e2e` script validates the complete ConfigMap workflow:

```bash
# Run full E2E test (snapshot → recipe → bundle)
./tools/e2e -s examples/snapshots/h100.yaml \
           -r examples/recipes/h100-eks-ubuntu-training.yaml \
           -b examples/bundles/h100-eks-ubuntu-training

# Just capture snapshot
./tools/e2e -s snapshot.yaml

# Generate recipe from ConfigMap
./tools/e2e -r recipe.yaml

# Get help
./tools/e2e --help
```

The e2e script:
1. Deploys agent Job to cluster
2. Waits for snapshot to be written to ConfigMap
3. Optionally saves snapshot to file
4. Optionally generates recipe using `cm://gpu-operator/cns-snapshot`
5. Optionally generates bundle from recipe
6. Validates each step completes successfully
```

### Adding a New Bundler

The bundler framework uses a **simplified RecipeResult-only architecture**. Bundlers receive RecipeResult with component references and generate deployment artifacts.

#### Quick Start: Minimal Bundler Implementation

1. Create bundler package in `pkg/component/<bundler-name>/`:
```go
// pkg/component/networkoperator/bundler.go
package networkoperator

import (
    "context"
    "embed"
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler"
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

const (
    bundlerType = bundler.BundleType("network-operator")
    Name        = "network-operator"  // Use constant for component name
)

func init() {
    // Self-register using MustRegister (panics on duplicates)
    bundler.MustRegister(bundlerType, NewBundler())
}

// Bundler generates Network Operator deployment bundles from RecipeResult.
type Bundler struct {
    *bundler.BaseBundler  // Embed helper for common functionality
}

// NewBundler creates a new Network Operator bundler instance.
func NewBundler() *Bundler {
    return &Bundler{
        BaseBundler: bundler.NewBaseBundler(bundlerType, templatesFS),
    }
}

// Make generates the bundle (delegates to makeFromRecipeResult).
func (b *Bundler) Make(ctx context.Context, input *result.RecipeResult, 
    outputDir string) (*bundler.Result, error) {
    return b.makeFromRecipeResult(ctx, input, outputDir)
}

// makeFromRecipeResult generates bundle from RecipeResult with component references.
func (b *Bundler) makeFromRecipeResult(ctx context.Context, input *result.RecipeResult, 
    outputDir string) (*bundler.Result, error) {
    
    // 1. Get component reference from RecipeResult
    component := input.GetComponentRef(Name)
    if component == nil {
        return nil, fmt.Errorf(Name + " component not found in recipe result")
    }
    
    // 2. Get values map (with overrides already applied)
    values := input.GetValuesForComponent(Name)
    
    // 3. Create bundle directory structure
    if err := b.CreateBundleDir(outputDir, "scripts"); err != nil {
        return nil, err
    }
    
    // 4. Generate script metadata
    scriptData := generateScriptData(component, values)
    
    // 5. Combine values and metadata for README
    readmeData := map[string]interface{}{
        "Values": values,
        "Script": scriptData,
    }
    
    // 6. Generate files from templates
    files := []struct {
        path     string
        template string
        data     interface{}
        perm     os.FileMode
    }{
        {filepath.Join(outputDir, "values.yaml"), "values.yaml", values, 0644},
        {filepath.Join(outputDir, "scripts/install.sh"), "install.sh", scriptData, 0755},
        {filepath.Join(outputDir, "README.md"), "README.md", readmeData, 0644},
    }
    
    var generatedFiles []string
    for _, f := range files {
        if err := b.GenerateFileFromTemplate(ctx, GetTemplate, f.template, 
            f.path, f.data, f.perm); err != nil {
            return nil, err
        }
        generatedFiles = append(generatedFiles, f.path)
    }
    
    // 7. Generate checksums and return result
    return b.GenerateResult(outputDir, generatedFiles)
}
```

2. Create script metadata generator in `scripts.go`:
```go
// pkg/component/networkoperator/scripts.go
package networkoperator

import (
    "time"
    
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
)

// ScriptData contains metadata for shell scripts and README.
type ScriptData struct {
    Timestamp     string
    Namespace     string
    Version       string
    Repository    string
    ComponentName string
}

// generateScriptData creates metadata from component reference.
func generateScriptData(component *result.ComponentRef, values map[string]interface{}) *ScriptData {
    namespace := "default"
    if ns, ok := values["namespace"].(string); ok {
        namespace = ns
    }
    
    repository := "https://helm.ngc.nvidia.com/nvidia"
    if repo, ok := values["repository"].(string); ok {
        repository = repo
    }
    
    return &ScriptData{
        Timestamp:     time.Now().Format(time.RFC3339),
        Namespace:     namespace,
        Version:       component.Version,
        Repository:    repository,
        ComponentName: component.Name,
    }
}
```

3. Create templates directory with embedded templates:
```
pkg/component/networkoperator/templates/
├── values.yaml.tmpl               # Helm chart values
├── install.sh.tmpl                # Installation script
├── uninstall.sh.tmpl              # Cleanup script
└── README.md.tmpl                 # Documentation
```

**Example template (`values.yaml.tmpl`):**
```yaml
# Network Operator Helm Values
# Generated by Cloud Native Stack

# Direct access to values map
version: {{ index . "version" }}
namespace: {{ index . "namespace" }}
  
driver:
  image: {{ index . "driver.image" }}
  version: {{ index . "driver.version" }}
  
config:
  rdma:
    enabled: {{ index . "rdma.enabled" }}
  sriov:
    enabled: {{ index . "sriov.enabled" }}
```

**Note:** Templates use `index` function to access values map.

4. Write tests with TestHarness and RecipeResult:
```go
// pkg/component/networkoperator/bundler_test.go
package networkoperator

import (
    "testing"
    
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
    "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
)

func TestBundler_Make(t *testing.T) {
    // Use TestHarness for consistent testing
    harness := internal.NewTestHarness(t, NewBundler())
    
    tests := []struct {
        name    string
        input   *result.RecipeResult
        wantErr bool
        verify  func(t *testing.T, outputDir string)
    }{
        {
            name:    "valid component reference",
            input:   createTestRecipeResult(),
            wantErr: false,
            verify: func(t *testing.T, outputDir string) {
                // TestHarness automatically verifies:
                // - All expected files exist
                // - Checksums are valid
                // - Directory structure is correct
                
                // Additional custom verification
                harness.AssertFileContains(outputDir, "values.yaml", 
                    "version:", "namespace:")
            },
        },
        {
            name:    "missing component reference",
            input:   &result.RecipeResult{},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := harness.RunTest(tt.input, tt.wantErr)
            if !tt.wantErr && tt.verify != nil {
                tt.verify(t, result.OutputDir)
            }
        })
    }
}

func createTestRecipeResult() *result.RecipeResult {
    return &result.RecipeResult{
        Components: map[string]*result.ComponentRef{
            Name: {
                Name:    Name,
                Version: "v25.4.0",
                Type:    "helm",
                Source:  "recipe",
                Values: map[string]interface{}{
                    "version":        "v25.4.0",
                    "namespace":      "network-operator",
                    "driver.image":   "nvcr.io/nvidia/mellanox/mofed",
                    "driver.version": "24.07",
                    "rdma.enabled":   true,
                    "sriov.enabled":  false,
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

# Test bundle generation (automatic registration via init())
./dist/cns_*/cnsctl bundle \
  --recipe examples/recipes/gb200-eks-ubuntu-training.yaml \
  --bundlers network-operator \
  --output ./test-bundles

# Verify bundle structure
tree test-bundles/network-operator/
```

**Expected output:**
```
test-bundles/network-operator/
├── values.yaml
├── scripts/
│   ├── install.sh
│   └── uninstall.sh
├── README.md
└── checksums.txt
```


#### Key Components

**RecipeResult-Only Architecture:**
- Single makeFromRecipeResult() path
- Get component via `input.GetComponentRef(Name)` 
- Extract values via `input.GetValuesForComponent(Name)`
- Values map already has CLI --set overrides applied
- No measurement extraction needed

**ScriptData for Metadata:**
- Contains namespace, version, repository, timestamp
- Separate from values map
- Used for shell scripts and README metadata

**Template Data:**
- values.yaml: receives values map directly
- scripts: receive ScriptData struct
- README.md: receives combined map with Values + Script

**File Structure per Component:**
- bundler.go: Main bundler logic with makeFromRecipeResult()
- scripts.go: ScriptData generation
- bundler_test.go: Tests using RecipeResult
- templates/*.tmpl: Embedded templates

#### Bundler Architecture Benefits

**Simplified Architecture:**
- 54% average code reduction across all bundlers
- Single code path (no dual Recipe/RecipeResult routing)
- Direct values access (no measurement extraction)
- Simpler testing with RecipeResult pattern

**BaseBundler Helper:**
- Common functionality: directory creation, file writing, template rendering, checksum generation
- Consistent error handling and logging
- Automatic context cancellation support

**TestHarness:**
- 34% less test code
- Consistent test structure across all bundlers
- Automatic file existence and checksum verification
- Helper assertions for common test patterns

**Registry Pattern:**
- Thread-safe bundler registration
- Self-registration via init() functions
- Automatic discovery (no manual registration needed)
- `MustRegister()` panics on duplicate types (fail fast)

#### Bundler Best Practices

**Implementation:**
- ✅ Use `Name` constant instead of hardcoded component names
- ✅ Single `makeFromRecipeResult()` method - no dual paths
- ✅ Get values via `input.GetValuesForComponent(Name)`
- ✅ Pass values map directly to templates
- ✅ Use `ScriptData` for metadata (namespace, version, timestamps)
- ✅ Use `go:embed` for template portability
- ✅ Keep bundlers stateless (thread-safe by default)
- ✅ Check context cancellation for long operations
- ✅ Use `MustRegister()` for fail-fast on registration errors

**Testing:**
- ✅ Use `TestHarness` for consistent test structure
- ✅ Create RecipeResult with ComponentRef in tests
- ✅ Test with realistic values maps
- ✅ Verify file content with `AssertFileContains()`
- ✅ Test error cases (missing component reference)
- ✅ Validate checksums are generated correctly

**Templates:**
- ✅ Access values map with `index` function: `{{ index . "key" }}`
- ✅ For README, use nested access: `{{ index .Values "key" }}`
- ✅ Use clear template variable names
- ✅ Add comments explaining data types
- ✅ Handle missing values gracefully with `{{- if }}`
- ✅ Validate template rendering in tests

**Documentation:**
- ✅ Add package doc.go with overview
- ✅ Document exported types and functions
- ✅ Include examples in README.md template
- ✅ Explain prerequisites and deployment steps


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
- **Three logging modes**:
  - **CLI Mode** (default): Minimal user-friendly output, just message text with red ANSI color for errors
  - **Text Mode** (--debug): Key=value format with full metadata (time, level, source, module, version)
  - **JSON Mode** (--log-json): Structured JSON format for machine parsing
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
    
    slog.Debug("request processed successfully", "id", id)
    return nil
}
```

**Logger Selection in CLI**:
```go
// In pkg/cli/root.go
switch {
case c.Bool("log-json"):
    logging.SetDefaultStructuredLoggerWithLevel(name, version, logLevel)
case isDebug:
    logging.SetDefaultLoggerWithLevel(name, version, logLevel)
default:
    logging.SetDefaultCLILogger(logLevel)  // Clean output for users
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

1. **Automated Checks** (GitHub Actions `on-push` workflow):
   - ✓ Go tests with race detector
   - ✓ golangci-lint (v2.6)
   - ✓ Trivy security scan (MEDIUM, HIGH, CRITICAL)
   - ✓ Code coverage upload to Codecov
   - Must pass before merge
2. **Maintainer Review**: A maintainer will review your code for:
   - Correctness and functionality
   - Code style and idioms
   - Test coverage and quality
   - Documentation completeness
3. **Feedback**: Address requested changes by pushing new commits
4. **Approval**: Once approved and CI passes, a maintainer will merge
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

## GitHub Actions & CI/CD

Cloud Native Stack uses a comprehensive CI/CD pipeline powered by GitHub Actions with a three-layer composite actions architecture.

### CI/CD Workflows

#### on-push.yaml (Continuous Integration)

**Trigger**: Push to `main` branch or pull requests to `main`

**Purpose**: Validate code quality on every commit/PR

**Steps**:
1. **Checkout Code** - Clone repository with full history
2. **Go CI Pipeline** - Uses `.github/actions/go-ci` composite action:
   - Setup Go (version 1.25)
   - Run tests with coverage
   - Upload coverage to Codecov (on main branch)
   - Run golangci-lint (version v2.6)
3. **Security Scan** - Uses `.github/actions/security-scan` composite action:
   - Trivy filesystem scan
   - Check for vulnerabilities (MEDIUM, HIGH, CRITICAL)
   - Upload SARIF results to GitHub Security tab

**Permissions**:
- `contents: read` - Read repository files
- `id-token: write` - OIDC token for attestations
- `security-events: write` - Upload security scan results

#### on-tag.yaml (Release Pipeline)

**Trigger**: Semantic version tags matching `v[0-9]+.[0-9]+.[0-9]+` (e.g., v0.8.12)

**Purpose**: Build, release, attest, and deploy production artifacts

**Steps**:
1. **Checkout Code** - Clone tagged release
2. **Go CI Pipeline** - Validate code before release (tests + lint)
3. **Build and Release** - Uses `.github/actions/go-build-release` composite action:
   - Authenticate to GHCR
   - Install build tools (ko, syft, crane, goreleaser)
   - Run `make release` (builds binaries + container images)
   - Generate binary SBOMs (SPDX format via GoReleaser)
   - Generate container SBOMs (SPDX format via Syft)
   - Publish to GitHub Releases and ghcr.io
4. **Attest Images** - Uses `.github/actions/attest-image-from-tag` composite action:
   - Resolve image digest from tag using crane
   - Generate SBOM attestations (Cosign)
   - Generate build provenance (GitHub Attestation API)
   - Sign with Sigstore keyless signing (Fulcio + Rekor)
   - Achieves **SLSA Build Level 3** compliance
5. **Deploy to Cloud Run** - Uses `.github/actions/cloud-run-deploy` composite action:
   - Authenticate using Workload Identity Federation (keyless)
   - Deploy cnsd to Google Cloud Run
   - Update service with new image version

**Permissions**:
- `attestations: write` - Generate attestations
- `contents: write` - Create GitHub releases
- `id-token: write` - OIDC authentication
- `packages: write` - Push to GHCR

### Composite Actions Architecture

Cloud Native Stack uses a **three-layer architecture** for maximum reusability:

#### Layer 1: Primitives (Single-Purpose Building Blocks)

- **ghcr-login** - GHCR authentication with github.token
- **setup-build-tools** - Modular tool installer (ko, syft, crane, goreleaser)
- **security-scan** - Trivy vulnerability scanning with SARIF upload

#### Layer 2: Composed Actions (Combine Primitives)

- **go-ci** - Complete Go CI pipeline:
  - Setup Go environment
  - Run tests with race detector
  - Upload coverage to Codecov (optional)
  - Run golangci-lint
  
- **go-build-release** - Full build and release pipeline:
  - Authenticate to GHCR (uses ghcr-login)
  - Install build tools (uses setup-build-tools)
  - Run `make release`
  - Output: `release_outcome` (success/failure)
  
- **attest-image-from-tag** - Resolve digest and generate attestations:
  - Install crane (uses setup-build-tools)
  - Authenticate to GHCR (uses ghcr-login)
  - Resolve digest from tag
  - Generate SBOM and provenance (uses sbom-and-attest)
  - Output: `image_digest`
  
- **sbom-and-attest** - Generate SBOM and attestations for known digest:
  - Install syft (uses setup-build-tools)
  - Generate SPDX SBOM
  - Sign with Cosign (keyless)
  - Generate GitHub attestation (provenance)
  
- **cloud-run-deploy** - Deploy to Google Cloud Run:
  - Authenticate with Workload Identity Federation
  - Deploy service with gcloud
  - Verify deployment

#### Layer 3: Workflows (Orchestrate Actions)

- **on-push.yaml** - CI validation for PRs and main branch
- **on-tag.yaml** - Release, attestation, and deployment

### Supply Chain Security

All releases include comprehensive supply chain security artifacts:

#### SLSA Build Provenance

- **Level**: SLSA Build Level 3
- **Format**: SLSA v1.0 attestation
- **Signing**: GitHub OIDC (keyless)
- **Transparency**: Rekor transparency log
- **Contents**:
  - Build trigger (tag push event)
  - Builder identity (GitHub Actions workflow)
  - Source repository and commit SHA
  - Workflow file path and run ID
  - Build parameters and environment
  - Resolved dependencies

#### SBOM Attestations

- **Binary SBOMs**: SPDX v2.3 format (GoReleaser)
- **Container SBOMs**: SPDX JSON format (Syft)
- **Signing**: Cosign keyless signing (Fulcio + Rekor)
- **Verification**: `gh attestation verify` or `cosign verify-attestation`
- **Contents**:
  - All Go module dependencies
  - Transitive dependencies
  - Package licenses (SPDX identifiers)
  - Package URLs (purl)
  - Container base image layers

#### Verification

```bash
# Get latest release tag
export TAG=$(curl -s https://api.github.com/repos/NVIDIA/cloud-native-stack/releases/latest | jq -r '.tag_name')

# Verify image attestations (GitHub CLI - Recommended)
gh attestation verify oci://ghcr.io/nvidia/cns:${TAG} --owner nvidia

# Verify with Cosign
cosign verify-attestation \
  --type spdxjson \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp 'https://github.com/NVIDIA/cloud-native-stack/.github/workflows/.*' \
  ghcr.io/nvidia/cns:${TAG}
```

For complete verification instructions, see [SECURITY.md](SECURITY.md).

### Best Practices

#### For Contributors

1. **Test Locally First**:
   ```bash
   make qualify  # Run tests + lint + scan (same as CI)
   ```

2. **Use Composite Actions**:
   - Reuse existing actions when possible
   - Follow single responsibility principle
   - Document inputs/outputs clearly

3. **Security**:
   - Pin external actions by SHA, not tag
   - Use minimal required permissions
   - Never log secrets or tokens
   - Use OIDC for cloud authentication (no stored credentials)

4. **Testing Actions**:
   ```bash
   # Test locally with act (GitHub Actions runner)
   act -j validate --secret GITHUB_TOKEN="$GITHUB_TOKEN"
   ```

#### For Maintainers

1. **Release Process**:
   ```bash
   # Create semantic version tag
   git tag v0.9.0
   git push origin v0.9.0
   
   # GitHub Actions automatically:
   # 1. Runs tests and lints
   # 2. Builds binaries and images
   # 3. Generates SBOMs and attestations
   # 4. Publishes to GitHub Releases and GHCR
   # 5. Deploys to Cloud Run
   ```

2. **Monitoring**:
   - Check GitHub Actions tab for workflow status
   - Review Security tab for vulnerability scan results
   - Monitor Cloud Run deployment health
   - Verify attestations after release

3. **Troubleshooting**:
   - Enable debug logging: Set `ACTIONS_STEP_DEBUG=true` in repo secrets
   - Re-run failed jobs from GitHub Actions UI
   - Check composite action logs for detailed errors
   - Verify OIDC token claims for authentication issues

For detailed GitHub Actions architecture documentation, see [.github/actions/README.md](.github/actions/README.md).

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
./dist/cns_*/cnsctl --debug snapshot

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
- [docs/MIGRATION.md](docs/MIGRATION.md) - Migration guide from v1 to v2
- [~archive/cns-v1/install-guides](~archive/cns-v1/install-guides) - Platform-specific installation (archived)
- [~archive/cns-v1/playbooks](~archive/cns-v1/playbooks) - Ansible automation guides (archived)
- [~archive/cns-v1/optimizations](~archive/cns-v1/optimizations) - Performance tuning guides (archived)
- [~archive/cns-v1/troubleshooting](~archive/cns-v1/troubleshooting) - Common issues and solutions (archived)

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

