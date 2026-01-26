# Claude Code Instructions

## Behavioral Constraints

- Be explicit and literal
- Prefer concrete examples over abstractions
- State uncertainty when present
- Concise over verbose
- Always identify: edge cases, failure modes, operational risks

## Anti-Patterns (Do Not Do)

| Anti-Pattern | Correct Approach |
|--------------|------------------|
| Modify code without reading it first | Always `Read` files before `Edit` |
| Skip or disable tests to make CI pass | Fix the actual issue |
| Invent new patterns | Study existing code in same package first |
| Use `fmt.Errorf` for errors | Use `pkg/errors` with error codes |
| Ignore context cancellation | Always check `ctx.Done()` in loops/operations |
| Add features not requested | Implement exactly what was asked |
| Create new files when editing suffices | Prefer `Edit` over `Write` |
| Guess at missing parameters | Ask for clarification |
| Continue after 3 failed fix attempts | Stop, reassess approach, explain blockers |

## Non-Negotiable Rules

1. **Read before writing** — Never modify code you haven't read
2. **Tests must pass** — `make test` with race detector; never skip tests
3. **Use project patterns** — Learn existing code before inventing new approaches
4. **3-strike rule** — After 3 failed fix attempts, stop and reassess
5. **Structured errors** — Use `pkg/errors` with error codes
6. **Context timeouts** — All I/O operations need context with timeout

## Project Overview

NVIDIA Cloud Native Stack (CNS) generates validated GPU-accelerated Kubernetes configurations.

**Workflow:** Snapshot → Recipe → Validate → Bundle

```
┌─────────┐    ┌────────┐    ┌──────────┐    ┌────────┐
│Snapshot │───▶│ Recipe │───▶│ Validate │───▶│ Bundle │
└─────────┘    └────────┘    └──────────┘    └────────┘
   │              │               │              │
   ▼              ▼               ▼              ▼
 Capture       Generate        Check         Create
 cluster       optimized      constraints    Helm values,
 state         config         vs actual     manifests
```

**Tech Stack:** Go 1.25, Kubernetes 1.33+, golangci-lint v2.6, Ko for images

**Key Packages:**

| Package | Purpose | Business Logic? |
|---------|---------|-----------------|
| `pkg/cli` | User interaction, input validation, output formatting | No |
| `pkg/api` | REST API handlers | No |
| `pkg/recipe` | Recipe resolution and overlay system | Yes |
| `pkg/bundler` | Artifact generation framework | Yes |
| `pkg/component/*` | Individual bundlers (GPU Operator, Network Operator, etc.) | Yes |
| `pkg/collector` | System state collection | Yes |
| `pkg/validator` | Constraint evaluation | Yes |
| `pkg/errors` | Structured error handling with codes | Yes |
| `pkg/k8s/client` | Singleton Kubernetes client | Yes |

**Critical Architecture Principle:**
- `pkg/cli` and `pkg/api` = user interaction only, no business logic
- Business logic lives in functional packages so CLI and API can both use it

## Required Patterns

**Errors (always use pkg/errors):**
```go
import "github.com/NVIDIA/cloud-native-stack/pkg/errors"

// Simple error
return errors.New(errors.ErrCodeNotFound, "GPU not found")

// Wrap existing error
return errors.Wrap(errors.ErrCodeInternal, "collection failed", err)

// With context
return errors.WrapWithContext(errors.ErrCodeTimeout, "operation timed out", ctx.Err(),
    map[string]interface{}{"component": "gpu-collector", "timeout": "10s"})
```

**Error Codes:** `ErrCodeNotFound`, `ErrCodeUnauthorized`, `ErrCodeTimeout`, `ErrCodeInternal`, `ErrCodeInvalidRequest`, `ErrCodeUnavailable`

**Context with timeout (always):**
```go
// Collectors: 10s timeout
func (c *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    // ...
}

// HTTP handlers: 30s timeout
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()
    // ...
}
```

**Table-driven tests (required for multiple cases):**
```go
func TestFunction(t *testing.T) {
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
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

**Functional options (configuration):**
```go
builder := recipe.NewBuilder(
    recipe.WithVersion(version),
)
server := server.New(
    server.WithName("cnsd"),
    server.WithVersion(version),
)
```

**Concurrency (errgroup):**
```go
g, ctx := errgroup.WithContext(ctx)
g.Go(func() error { return collector1.Collect(ctx) })
g.Go(func() error { return collector2.Collect(ctx) })
if err := g.Wait(); err != nil {
    return fmt.Errorf("collection failed: %w", err)
}
```

**Structured logging (slog):**
```go
slog.Debug("request started", "requestID", requestID, "method", r.Method)
slog.Error("operation failed", "error", err, "component", "gpu-collector")
```

## Common Tasks

| Task | Location | Key Points |
|------|----------|------------|
| New bundler | `pkg/component/<name>/` | Embed `BaseBundler`, implement `Make()`, self-register with `MustRegister()` |
| New collector | `pkg/collector/<type>/` | Implement `Collector` interface, add to factory |
| New API endpoint | `pkg/api/` | Handler + middleware chain + OpenAPI spec update |
| Fix test failures | Run `make test` | Check race conditions (`-race`), verify context handling |

**Bundler template (use pkg/component/gpuoperator as reference):**
```go
func init() {
    bundler.MustRegister(bundlerType, NewBundler())
}

type Bundler struct {
    *bundler.BaseBundler
}

func (b *Bundler) Make(ctx context.Context, input *result.RecipeResult, outputDir string) (*bundler.Result, error) {
    component := input.GetComponentRef(Name)
    values := input.GetValuesForComponent(Name)
    // Generate files...
    return b.GenerateResult(outputDir, generatedFiles)
}
```

## Commands

```bash
# Development
make qualify      # Full check: test + lint + scan (run before PR)
make test         # Unit tests with -race
make lint         # golangci-lint + yamllint
make scan         # Trivy security scan
make build        # Build binaries
make tidy         # Format + update deps

# CLI workflow
cnsctl snapshot --output snapshot.yaml
cnsctl recipe --snapshot snapshot.yaml --intent training --output recipe.yaml
cnsctl bundle --recipe recipe.yaml --bundlers gpu-operator --output ./bundles
cnsctl validate --recipe recipe.yaml --snapshot snapshot.yaml

# With overrides
cnsctl bundle -r recipe.yaml -b gpu-operator \
  --set gpuoperator:driver.version=570.86.16 \
  --deployer argocd \
  -o ./bundles
```

## Decision Framework

When choosing between approaches, prioritize in this order:
1. **Testability** — Can it be unit tested without external dependencies?
2. **Readability** — Can another engineer understand it quickly?
3. **Consistency** — Does it match existing patterns in the codebase?
4. **Simplicity** — Is it the simplest solution that works?
5. **Reversibility** — Can it be easily changed later?

## Troubleshooting

| Issue | Check |
|-------|-------|
| K8s connection fails | `~/.kube/config` or `KUBECONFIG` env |
| GPU not detected | `nvidia-smi` in PATH |
| Linter errors | Use `errors.Is()` not `==`; add `return` after `t.Fatal()` |
| Race conditions | Run with `-race` flag |
| Build failures | Run `make tidy` |

## Key Files

| File | Purpose |
|------|---------|
| `pkg/recipe/data/data-v1.yaml` | Recipe overlay definitions |
| `api/cns/v1/server.yaml` | OpenAPI spec |
| `.goreleaser.yaml` | Release configuration |
| `go.mod` | Dependencies |

## Full Reference

See `.github/copilot-instructions.md` for extended documentation:
- Detailed code examples for collectors, bundlers, API endpoints
- GitHub Actions architecture (three-layer composite actions)
- CI/CD workflows (on-push.yaml, on-tag.yaml)
- Supply chain security (SLSA, SBOM, Cosign)
- E2E testing patterns
- ConfigMap-based workflows
