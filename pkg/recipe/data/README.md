# Recipe Data Directory

This directory contains recipe metadata and component configurations for the NVIDIA Cloud Native Stack bundler system.

## Quick Reference

| Task | Documentation |
|------|--------------|
| Understand recipe architecture | [Data Architecture](../../../docs/architecture/data.md) |
| Create/modify recipes | [Recipe Development Guide](../../../docs/integration/recipe-development.md) |
| Create new bundlers | [Bundler Development Guide](../../../docs/architecture/component.md) |
| CLI commands | [CLI Reference](../../../docs/user-guide/cli-reference.md) |

## Directory Structure

```
pkg/recipe/data/
├── base.yaml                    # Base recipe (universal defaults)
├── eks-gb200-training.yaml      # EKS + GB200 training overlay
├── ...
└── components/                  # Component value configurations
    ├── cert-manager/
    ├── gpu-operator/
    ├── ...
```

## Overview

The recipe system uses a **base-plus-overlay architecture**:

- **Base values** (`base.yaml`) provide default configurations
- **Overlay values** (e.g., `eks-gb200-training.yaml`) provide environment-specific optimizations
- **Inline overrides** allow per-recipe customization without creating new files

All files in this directory are embedded into the CLI binary and API server at compile time.

### Run Validation Tests

```bash
# Run all recipe tests
make test

# Run specific validation
go test -v ./pkg/recipe/... -run TestAllMetadataFilesConformToSchema

# Check for duplicate criteria
go test -v ./pkg/recipe/... -run TestNoDuplicateCriteriaAcrossOverlays
```

## Automated Validation

All recipe metadata and component values are automatically validated. Tests run as part of `make test` and check:

- Schema conformance (YAML parses correctly)
- Criteria validation (valid enum values)
- Reference validation (valuesFile paths exist, dependencyRefs resolve)
- Constraint syntax (valid measurement paths and operators)
- Uniqueness (no duplicate criteria across overlays)
- Merge consistency (base + overlay merges without data loss)

```bash
# Generate bundle from recipe with overrides
cnsctl bundle -r pkg/recipe/data/your-recipe.yaml -o ./test-bundles

# Verify merged values
cat test-bundles/gpu-operator/values.yaml | grep -A5 "driver:"
```
For detailed test documentation, see [Automated Validation](../../../docs/architecture/data.md#automated-validation).

## See Also

- [Data Architecture](../../../docs/architecture/data.md) - Recipe generation process, overlay system, query matching
- [Recipe Development Guide](../../../docs/integration/recipe-development.md) - How to create and modify recipes
- [Bundler Development Guide](../../../docs/architecture/component.md) - How to create new bundlers
