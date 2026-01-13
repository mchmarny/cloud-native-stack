# Recipe Data Directory

This directory contains recipe metadata and component configurations for the NVIDIA Cloud Native Stack bundler system.

## Overview

The recipe system uses a **base-plus-overlay architecture** where:
- **Base values** (`base.yaml`) provide default configurations
- **Overlay values** (e.g., `eks-gb200-training.yaml`) provide environment-specific optimizations
- **Inline overrides** allow per-recipe customization without creating new files

## Component Value Configuration Patterns

The bundler supports three patterns for configuring component values, each optimized for different use cases:

### Pattern 1: ValuesFile Only (Basic)

All configuration comes from a separate values file. Best for large configurations that are reusable across multiple recipes.

```yaml
componentRefs:
  - name: cert-manager
    type: Helm
    version: v1.17.3
    valuesFile: components/cert-manager/eks-values.yaml
    # No overrides - everything in the file
```

**When to use:**
- Large configurations (100+ lines)
- Reusable across multiple recipes
- Team collaboration with clear file ownership
- Separate overlay files already exist

### Pattern 2: Overrides Only (Self-Contained)

All configuration is inline in the recipe - no separate values file needed. Best for small configurations or recipe-specific deployments.

```yaml
componentRefs:
  - name: nvsentinel
    type: Helm
    version: v0.6.0
    # Note: No valuesFile specified
    overrides:
      namespace: nvsentinel
      sentinel:
        enabled: true
        logLevel: info
        metrics:
          enabled: true
      resources:
        limits:
          memory: 256Mi
        requests:
          cpu: 100m
          memory: 128Mi
```

**When to use:**
- Small configurations (<50 lines)
- Unique, recipe-specific settings
- One-off deployments or testing
- Self-contained recipes (no external dependencies)

### Pattern 3: Hybrid (ValuesFile + Overrides)

Base configuration in a values file, with recipe-specific tweaks as inline overrides. Best for large shared configurations with small per-recipe customizations.

```yaml
componentRefs:
  # Example 1: Override just one field
  - name: gpu-operator
    type: Helm
    version: v25.3.4
    valuesFile: components/gpu-operator/eks-gb200-training.yaml
    overrides:
      # Override just the driver version for this specific deployment
      driver:
        version: "570.86.16"
      # Add deployment-specific feature flag not in base file
      experimental:
        newFeature: true

  # Example 2: Override multiple sections with deep merge
  - name: network-operator
    type: Helm
    version: v25.4.0
    valuesFile: components/network-operator/values.yaml
    overrides:
      # Override operator configuration
      operator:
        repository: nvcr.io/custom-registry
        tag: v25.4.0-custom
      # Override RDMA settings
      rdma:
        enabled: true
        useHostMofed: false
      # Add new field not in base values
      sriov:
        enabled: true
        numVfs: 8
```

**When to use:**
- Large base configuration with small recipe-specific tweaks
- Environment-specific overrides (dev/staging/prod)
- Version pinning per deployment
- Feature flags or experimental settings

## Value Merge Precedence

Values are merged in this order (later sources override earlier ones):

```
Base Values (lowest precedence)
    ↓
ValuesFile (overlay)
    ↓
Overrides (highest precedence)
```

**Deep merge behavior:**
- Only specified fields in overrides are replaced
- Unspecified fields are preserved from base/ValuesFile
- New fields in overrides are added to the final configuration
- Arrays are replaced entirely (not merged element-by-element)
  
> Note: user has the last word, can override final recipe state with `--set` flags on `cnsctl bundle`.

**Example:**

Base values (`components/gpu-operator/base.yaml`):
```yaml
driver:
  version: "550.54.15"
  repository: nvcr.io/nvidia
  image: driver
gds:
  enabled: false
```

Overlay values (`components/gpu-operator/eks-gb200-training.yaml`):
```yaml
driver:
  version: "570.86.16"  # Override
gds:
  enabled: true         # Override
```

Recipe with inline overrides:
```yaml
valuesFile: components/gpu-operator/eks-gb200-training.yaml
overrides:
  driver:
    version: "580.13.01"  # Override again
```

**Final merged result:**
```yaml
driver:
  version: "580.13.01"      # From inline override (highest)
  repository: nvcr.io/nvidia  # From base (preserved)
  image: driver               # From base (preserved)
gds:
  enabled: true              # From overlay valuesFile
```

## File Naming Conventions

- `base.yaml` - Base values for a component (defaults, common settings)
- `{service}-{gpu}-{os}-{intent}.yaml` - Overlay values for specific environment
  - Examples: `eks-gb200-ubuntu-training.yaml`, `gke-h100-ubuntu-inference.yaml`
- Recipe metadata files use overlay naming for easy discovery


## Testing

Test your recipes with inline overrides:

```bash
# Generate bundle from recipe with overrides
cnsctl bundle -r pkg/recipe/data/your-recipe.yaml -o ./test-bundles

# Verify merged values
cat test-bundles/gpu-operator/values.yaml | grep -A5 "driver:"
```

## See Also

- [Bundler Development Guide](../../../docs/architecture/component.md)
- [Recipe Architecture](../../../docs/architecture/data.md)
- [CLI Reference](../../../docs/user-guide/cli-reference.md)
