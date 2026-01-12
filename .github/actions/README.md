# GitHub Actions Architecture

This directory contains a modular, reusable GitHub Actions architecture optimized for separation of concerns and composability.

## Composite Actions

### Core CI/CD Actions

#### `go-ci/`
**Purpose**: Complete Go project CI pipeline (setup, test, lint)  
**When to use**: Every workflow that needs to validate Go code  
**Inputs**:
- `go_version` (required): Go version (e.g., "1.25")
- `golangci_lint_version` (required): golangci-lint version (e.g., "v2.6")
- `upload_codecov` (optional): Whether to upload coverage to Codecov (default: "false")

**Example**:
```yaml
- uses: ./.github/actions/go-ci
  with:
    go_version: '1.25'
    golangci_lint_version: 'v2.6'
    upload_codecov: 'true'
```

#### `security-scan/`
**Purpose**: Trivy vulnerability scanning with SARIF upload  
**When to use**: Security validation in CI/CD pipelines  
**Inputs**:
- `scan_type` (optional): Scan type (default: "fs")
- `scan_ref` (optional): Scan target (default: ".")
- `severity` (optional): Severity levels (default: "HIGH,CRITICAL")
- `output_file` (optional): SARIF file name (default: "trivy-results.sarif")
- `category` (optional): GitHub Security category (default: "trivy")
- `skip_dirs` (optional): Directories to skip (default: "")

**Example**:
```yaml
- uses: ./.github/actions/security-scan
  with:
    severity: 'MEDIUM,HIGH,CRITICAL'
    category: 'trivy-fs'
```

### Build & Release Actions

#### `setup-build-tools/`
**Purpose**: Install container build tools (ko, syft, crane, goreleaser)  
**When to use**: When you need specific build tools without full build pipeline  
**Inputs**:
- `install_ko` (optional): Install ko (default: "false")
- `install_syft` (optional): Install syft (default: "false")
- `install_crane` (optional): Install crane (default: "false")
- `crane_version` (optional): crane version (default: "v0.20.6")
- `install_goreleaser` (optional): Install goreleaser (default: "false")

**Example**:
```yaml
- uses: ./.github/actions/setup-build-tools
  with:
    install_ko: 'true'
    install_crane: 'true'
    crane_version: 'v0.20.6'
```

#### `go-build-release/`
**Purpose**: Complete build and release pipeline (tools + auth + make release)  
**When to use**: Release workflows that build and publish artifacts  
**Inputs**:
- `registry` (optional): Container registry (default: "ghcr.io")
- `ko_docker_repo` (optional): KO_DOCKER_REPO override (default: "")

**Outputs**:
- `release_outcome`: Release step outcome (success/failure)

**Example**:
```yaml
- uses: ./.github/actions/go-build-release
  id: release
- if: steps.release.outputs.release_outcome == 'success'
  run: echo "Release succeeded"
```

### Attestation Actions

#### `ghcr-login/`
**Purpose**: Authenticate to GitHub Container Registry  
**When to use**: Before any GHCR operations (shared authentication)  
**Inputs**:
- `registry` (optional): Registry URL (default: "ghcr.io")
- `username` (optional): Username (default: github.actor)

**Example**:
```yaml
- uses: ./.github/actions/ghcr-login
```

#### `attest-image-from-tag/`
**Purpose**: Resolve digest from tag and generate SBOM + provenance  
**When to use**: Attesting images by tag (typical release workflow)  
**Inputs**:
- `image_name` (required): Full image name without tag (e.g., "ghcr.io/org/image")
- `tag` (required): Image tag (e.g., "v1.2.3")
- `crane_version` (optional): crane version (default: "v0.20.6")

**Outputs**:
- `image_digest`: Resolved sha256 digest

**Example**:
```yaml
- uses: ./.github/actions/attest-image-from-tag
  with:
    image_name: ghcr.io/${{ github.repository_owner }}/my-app
    tag: ${{ github.ref_name }}
```

#### `sbom-and-attest/`
**Purpose**: Generate SBOM and attestations for image with known digest  
**When to use**: When you already have the digest (e.g., from build output)  
**Inputs**:
- `image_name` (required): Full image name
- `image_digest` (required): sha256 digest

**Example**:
```yaml
- uses: ./.github/actions/sbom-and-attest
  with:
    image_name: ghcr.io/org/image
    image_digest: sha256:abc123...
```

### Deployment Actions

#### `cloud-run-deploy/`
**Purpose**: Deploy to Google Cloud Run with Workload Identity  
**When to use**: Cloud Run deployments from CI/CD  
**Inputs**:
- `project_id` (required): GCP project ID
- `workload_identity_provider` (required): WIF provider resource name
- `service_account` (required): Service account email
- `region` (required): Cloud Run region
- `service` (required): Cloud Run service name
- `image` (required): Container image reference

**Example**:
```yaml
- uses: ./.github/actions/cloud-run-deploy
  with:
    project_id: 'my-project'
    workload_identity_provider: 'projects/.../providers/github'
    service_account: 'deployer@my-project.iam.gserviceaccount.com'
    region: 'us-west1'
    service: 'api'
    image: 'ghcr.io/org/api:v1.0.0'
```

## Workflows

### `on-push.yaml`
**Trigger**: Push to main, PRs to main  
**Purpose**: CI validation  
**Steps**:
1. Checkout
2. Go CI (setup, test, lint)
3. Security scan

### `on-tag.yaml`
**Trigger**: Semantic version tags (v*.*.*)  
**Purpose**: Build, release, attest, deploy  
**Steps**:
1. Checkout
2. Go CI (setup, test, lint)
3. Build and release
4. Attest images (cnsd, cnsctl)
5. Deploy to Cloud Run

## Architecture Principles

### Separation of Concerns
- **Single Responsibility**: Each action does one thing well
- **Composability**: Actions can be combined for complex workflows
- **Testability**: Small actions are easier to test in isolation

### Reusability Layers
1. **Primitive Actions**: Low-level operations (ghcr-login, setup-build-tools)
2. **Composed Actions**: Combine primitives (attest-image-from-tag = login + crane + sbom-and-attest)
3. **Pipeline Actions**: Full workflows (go-build-release = tools + auth + release)

### Authentication Strategy
- GHCR authentication centralized in `ghcr-login` action
- All actions requiring registry access use this shared action
- Eliminates redundant login steps (was happening 3x in on-tag workflow)

### Tool Installation Strategy
- Build tools centralized in `setup-build-tools` action
- Selective installation via boolean flags reduces overhead
- Version pinning ensures reproducibility

## Migration from Previous Architecture

### Removed Redundancies
- **Before**: 3 separate GHCR logins (attest-image-from-tag, sbom-and-attest, workflow)
- **After**: Single `ghcr-login` action reused everywhere

- **Before**: Inline Trivy scan + upload steps in workflow
- **After**: Reusable `security-scan` action

- **Before**: 4 separate tool installations in workflow (ko, syft, crane, goreleaser)
- **After**: Single `go-build-release` or selective `setup-build-tools`

### Benefits
- **Less Code**: ~40% reduction in workflow YAML
- **Better Reuse**: Actions portable to other repos/workflows
- **Clearer Intent**: Pipeline steps self-document through action names
- **Easier Testing**: Individual actions can be tested independently
- **Version Management**: Tool versions centralized in action defaults

## Adding New Workflows

### For a simple CI workflow
```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: ./.github/actions/go-ci
        with:
          go_version: '1.25'
          golangci_lint_version: 'v2.6'
      - uses: ./.github/actions/security-scan
```

### For a release workflow with attestations
```yaml
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: ./.github/actions/go-ci
        with:
          go_version: '1.25'
          golangci_lint_version: 'v2.6'
      - uses: ./.github/actions/go-build-release
        id: release
      - uses: ./.github/actions/attest-image-from-tag
        with:
          image_name: ghcr.io/org/app
          tag: ${{ github.ref_name }}
```

### For custom tool combinations
```yaml
steps:
  - uses: ./.github/actions/setup-build-tools
    with:
      install_crane: 'true'
      install_ko: 'true'
  - run: |
      ko build ./cmd/my-app
      crane digest ghcr.io/org/my-app:latest
```

## Future Enhancements

### Potential Improvements
1. **Matrix Attestation Action**: Accept arrays of images to attest N images in one step
2. **Reusable Workflow**: For full "CI → release → attest → deploy" as a callable workflow
3. **Multi-Registry Support**: Extend ghcr-login to support DockerHub, ECR, GAR, etc.
4. **Parallel Attestations**: Run attestations concurrently for faster builds
5. **Cache Management**: Centralized Go module/build cache management action
6. **Notification Action**: Slack/Discord/PagerDuty notifications for workflow events

### Cross-Repo Reusability
To use these actions in other repositories:
```yaml
- uses: NVIDIA/cloud-native-stack/.github/actions/go-ci@main
  with:
    go_version: '1.25'
    golangci_lint_version: 'v2.6'
```
