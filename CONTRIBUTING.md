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
│   ├── api/                 # HTTP server and handlers
│   ├── cli/                 # CLI commands (urfave/cli)
│   ├── client/              # API client library
│   ├── collector/           # System configuration collectors
│   ├── logging/             # Structured logging utilities
│   ├── measurement/         # Performance measurements
│   ├── node/                # Node information
│   ├── recipe/              # Recipe generation logic
│   ├── serializer/          # Output formatting (JSON/YAML/table)
│   ├── server/              # Server configuration
│   ├── snapshotter/         # Snapshot orchestration
│   └── version/             # Version information
├── deployments/             # Kubernetes manifests
│   └── eidos-agent/         # Agent Job and RBAC
├── docs/                    # Documentation
│   ├── install-guides/      # Platform-specific guides
│   ├── playbooks/           # Ansible automation
│   ├── optimizations/       # Performance tuning
│   └── troubleshooting/     # Common issues
├── examples/                # Example configurations
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
- **Purpose**: User-facing tool for system snapshots and recipe generation

#### API Server
- **Location**: `cmd/eidos-api-server/main.go` → `pkg/api/`
- **Endpoints**: `/v1/recipe`, `/healthz`
- **Purpose**: HTTP service for recipe generation

#### Collectors
- **Location**: `pkg/collector/`
- **Pattern**: Factory-based registration
- **Types**: SystemD, OS (grub, sysctl, kmod, release), Kubernetes, GPU
- **Purpose**: Gather system configuration data
- **OS Release Collector**: New 4th OS subtype that captures `/etc/os-release` (ID, VERSION_ID, PRETTY_NAME, etc.)

#### Recipe Engine
- **Location**: `pkg/recipe/`
- **Purpose**: Generate optimized configurations based on environment parameters
- **Input**: OS, kernel, GPU type, service type, workload intent
- **Output**: Configuration recommendations

### Common Make Targets

```bash
# Development
make tidy         # Format code and update dependencies
make build        # Build binaries for current platform
make server       # Start API server locally (debug mode)

# Testing
make test         # Run unit tests with coverage
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
make bump-major   # Bump major version
make bump-minor   # Bump minor version
make bump-patch   # Bump patch version
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

# Snapshot with debug logging
./dist/eidos_*/eidos --debug snapshot --format yaml

# Recipe generation
./dist/eidos_*/eidos recipe \
  --os ubuntu \
  --osv 24.04 \
  --service eks \
  --gpu h100 \
  --format yaml
```

### Running the API Server

```bash
# Development mode with debug logging
make server

# Custom configuration
PORT=8080 LOG_LEVEL=debug go run cmd/eidos-api-server/main.go

# Test endpoints
curl http://localhost:8080/healthz
curl "http://localhost:8080/v1/recipe?os=ubuntu&gpu=h100"
```

### Testing in Kubernetes

Build and deploy the agent locally:

```bash
# Build container image
ko build --local ./cmd/eidos-api-server

# Update deployment with local image
kubectl set image job/eidos -n gpu-operator eidos=<local-image>

# Deploy and check logs
kubectl apply -f deployments/eidos-agent/
kubectl logs -n gpu-operator job/eidos -f
```

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

- **Coverage**: Aim for meaningful test coverage (current: ~28%, target: >50%)
- **Unit tests**: Test all public functions and methods
- **Table-driven tests**: Use for multiple test cases
- **Error cases**: Test error conditions and edge cases
- **Mocks**: Mock external dependencies (filesystem, network, etc.)

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
- Standard library for most functionality

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

