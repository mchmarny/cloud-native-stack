# Bundler Development Guide

Learn how to create new bundlers for Cloud Native Stack.

## Overview

Bundlers convert recipes into deployment artifacts. Artifacts include Helm values files, Kubernetes manifests, and installation scripts.

**Framework features:**

- **BaseBundler**: Helper struct providing common operations (directory creation, file writing, template rendering, checksum generation)
- **Self-registration**: Bundlers register themselves via `init()` function using `bundler.MustRegister()`
- **Parallel execution**: Multiple bundlers run concurrently via goroutines with errgroup context cancellation
- **Recipe parsing**: `pkg/bundler/internal` package provides helper functions for extracting recipe measurements
- **Template system**: Bundlers embed templates using `go:embed` directive
- **Test infrastructure**: `TestHarness` struct standardizes bundler testing
- **E2E validation**: `tools/e2e` script tests complete workflow including bundle generation

**Implemented bundlers:**

- GPU Operator: Generates Helm values, ClusterPolicy manifest, installation scripts
- Network Operator: Generates Helm values, NICClusterPolicy manifest, network configuration
- Skyhook: Generates Helm values, Skyhook CR manifest, node optimization scripts

## Quick Start

### Minimal Bundler Implementation

```go
// pkg/bundler/mybundler/bundler.go
package mybundler

import (
    "context"
    "embed"
    "path/filepath"
    
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler"
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
    "github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

const bundlerType = bundler.BundleType("my-bundler")

func init() {
    // Self-register (panics on duplicates)
    bundler.MustRegister(bundlerType, NewBundler())
}

// Bundler generates deployment bundles.
type Bundler struct {
    *bundler.BaseBundler  // Embed helper
}

// NewBundler creates a new bundler instance.
func NewBundler() *Bundler {
    return &Bundler{
        BaseBundler: bundler.NewBaseBundler(bundlerType, templatesFS),
    }
}

// Make generates the bundle (override BaseBundler.Make).
func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, 
    outputDir string) (*bundler.BundleResult, error) {
    
    // 1. Create bundle directory structure
    dirs := []string{"manifests", "scripts"}
    if err := b.CreateBundleDir(outputDir, dirs...); err != nil {
        return nil, err
    }
    
    // 2. Build configuration map from recipe
    configMap := b.buildConfigMap(r)
    
    // 3. Generate typed data structures from recipe
    helmValues := GenerateHelmValues(r, configMap)
    scriptData := GenerateScriptData(r, configMap)
    readmeData := map[string]interface{}{
        "Helm":   helmValues,
        "Script": scriptData,
    }
    
    // 4. Generate files from templates
    files := []struct{path, template string; data interface{}; perm int}{
        {filepath.Join(outputDir, "values.yaml"), "values.yaml", helmValues, 0644},
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
    
    // 5. Generate checksums and return result
    return b.GenerateResult(outputDir, generatedFiles)
}

func (b *Bundler) buildConfigMap(r *recipe.Recipe) map[string]string {
    // Build configuration map from recipe
    configMap := internal.BuildBaseConfigMap(r, nil)
    // Add bundler-specific extractions...
    return configMap
}
```

## Architecture

### BaseBundler Helper

BaseBundler provides common functionality:

```go
// Provided methods:
CreateBundleDir(path, subdirs...)     // Creates directory structure
WriteFile(path, content)               // Writes file with error handling
GenerateFileFromTemplate(...)          // Renders struct to template
GenerateResult(dir, files)             // Creates BundleResult with checksums
Type()                                 // Returns bundler type
Validate(ctx, recipe)                  // Default validation (override if needed)
```

### Component Structure

```
pkg/bundler/mybundler/
├── bundler.go          # Main bundler logic
├── helm.go             # Helm values generation
├── manifests.go        # Manifest generation
├── scripts.go          # Script generation  
├── bundler_test.go     # Tests
└── templates/          # Embedded templates
    ├── values.yaml.tmpl
    ├── manifest.yaml.tmpl
    ├── install.sh.tmpl
    └── README.md.tmpl
```

## Step-by-Step Guide

### Step 1: Create Package Structure

```shell
mkdir -p pkg/bundler/mybundler/templates
cd pkg/bundler/mybundler
```

### Step 2: Create Bundler Implementation

```go
// bundler.go
package mybundler

import (
    "context"
    "embed"
    "fmt"
    "path/filepath"
    
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler"
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
    "github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

const bundlerType = bundler.BundleType("my-bundler")

func init() {
    bundler.MustRegister(bundlerType, NewBundler())
}

type Bundler struct {
    *bundler.BaseBundler
}

func NewBundler() *Bundler {
    return &Bundler{
        BaseBundler: bundler.NewBaseBundler(bundlerType, templatesFS),
    }
}

// GetTemplate returns template content by name.
func GetTemplate(name string) ([]byte, error) {
    return templatesFS.ReadFile(fmt.Sprintf("templates/%s.tmpl", name))
}

func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, 
    outputDir string) (*bundler.BundleResult, error) {
    
    // Implementation here
    return b.generateBundle(ctx, r, outputDir)
}
```

### Step 3: Extract Configuration from Recipe

```go
// buildConfigMap extracts string values from recipe
func (b *Bundler) buildConfigMap(r *recipe.Recipe) map[string]string {
    // Start with base config
    configMap := internal.BuildBaseConfigMap(r, nil)
    
    // Extract K8s image versions
    imageSubtype := internal.ExtractK8sImageSubtype(r)
    if imageSubtype != nil {
        if version, exists := imageSubtype.Data["my-operator"]; exists {
            configMap["my_operator_version"] = version.String()
        }
    }
    
    // Extract GPU settings
    gpuSubtype := internal.ExtractGPUDeviceSubtype(r)
    if gpuSubtype != nil {
        if model, exists := gpuSubtype.Data["model"]; exists {
            configMap["gpu_model"] = model.String()
        }
    }
    
    // Extract registry info
    registrySubtype := internal.ExtractK8sRegistrySubtype(r)
    if registrySubtype != nil {
        if uri, exists := registrySubtype.Data["uri"]; exists {
            configMap["driver_registry"] = uri.String()
        }
    }
    
    return configMap
}
```

### Step 4: Generate Typed Data Structures

```go
// helm.go
package mybundler

import (
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/common"
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
    "github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// HelmValues contains typed configuration for Helm templates
type HelmValues struct {
    // Common fields
    Timestamp      string
    Version        string
    RecipeVersion  string
    
    // Operator-specific fields
    OperatorVersion    common.ValueWithContext
    DriverVersion      common.ValueWithContext
    EnableFeatureX     common.ValueWithContext
    
    // Registry
    DriverRegistry     common.ValueWithContext
}

// GenerateHelmValues creates typed HelmValues from recipe
func GenerateHelmValues(r *recipe.Recipe, config map[string]string) *HelmValues {
    // Build timestamp and versions
    helmValues := &HelmValues{
        Timestamp:     internal.GetCurrentTimestamp(),
        Version:       internal.GetVersion(),
        RecipeVersion: internal.GetRecipeVersion(r),
    }
    
    // Initialize with defaults and config values
    helmValues.OperatorVersion = common.ValueWithContext{
        Value:   internal.GetConfigValue(config, "my_operator_version", "v1.0.0"),
        Context: "",
    }
    
    helmValues.DriverRegistry = common.ValueWithContext{
        Value:   internal.GetConfigValue(config, "driver_registry", "nvcr.io/nvidia"),
        Context: "",
    }
    
    // Extract values with context from recipe
    if k8sImageSubtype := internal.ExtractK8sImageSubtype(r); k8sImageSubtype != nil {
        helmValues.OperatorVersion = internal.GetFieldWithContext(
            k8sImageSubtype, "my-operator", helmValues.OperatorVersion.Value)
    }
    
    return helmValues
}
```

### Step 5: Create Templates

```yaml
# templates/values.yaml.tmpl
# {{ .Timestamp }}
# Bundler Version: {{ .Version }}
# Recipe Version: {{ .RecipeVersion }}

myOperator:
  version: {{ .OperatorVersion.Value }}
  
  driver:
    image: {{ .DriverRegistry.Value }}/my-driver
    version: {{ .DriverVersion.Value }}
  
  features:
    featureX: {{ .EnableFeatureX.Value }}
```

```bash
# templates/install.sh.tmpl
#!/bin/bash
# Generated by CNS Eidos - {{ .Timestamp }}

set -e

echo "Installing My Operator..."

# Add Helm repository
helm repo add my-operator https://charts.example.com
helm repo update

# Install operator
helm install my-operator my-operator/my-operator \
  -n my-operator-system \
  --create-namespace \
  -f values.yaml

echo "Installation complete!"
```

### Step 6: Generate Bundle Files

```go
func (b *Bundler) generateBundle(ctx context.Context, r *recipe.Recipe, 
    outputDir string) (*bundler.BundleResult, error) {
    
    // 1. Create directories
    dirs := []string{"manifests", "scripts"}
    if err := b.CreateBundleDir(outputDir, dirs...); err != nil {
        return nil, err
    }
    
    // 2. Build config
    configMap := b.buildConfigMap(r)
    
    // 3. Generate data structures
    helmValues := GenerateHelmValues(r, configMap)
    scriptData := GenerateScriptData(r, configMap)
    
    // 4. Generate files
    var generatedFiles []string
    
    // Generate values.yaml
    valuesPath := filepath.Join(outputDir, "values.yaml")
    if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "values.yaml",
        valuesPath, helmValues, 0644); err != nil {
        return nil, err
    }
    generatedFiles = append(generatedFiles, valuesPath)
    
    // Generate install script
    installPath := filepath.Join(outputDir, "scripts/install.sh")
    if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "install.sh",
        installPath, scriptData, 0755); err != nil {
        return nil, err
    }
    generatedFiles = append(generatedFiles, installPath)
    
    // 5. Generate result with checksums
    return b.GenerateResult(outputDir, generatedFiles)
}
```

### Step 7: Write Tests

```go
// bundler_test.go
package mybundler

import (
    "testing"
    
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
    "github.com/NVIDIA/cloud-native-stack/pkg/measurement"
    "github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

func TestBundler_Make(t *testing.T) {
    // Use TestHarness for consistent testing
    harness := internal.NewTestHarness(t, NewBundler())
    
    tests := []struct {
        name    string
        recipe  *recipe.Recipe
        wantErr bool
        verify  func(t *testing.T, outputDir string)
    }{
        {
            name:    "valid recipe",
            recipe:  createTestRecipe(),
            wantErr: false,
            verify: func(t *testing.T, outputDir string) {
                // TestHarness automatically verifies:
                // - All expected files exist
                // - Checksums are valid
                // - Directory structure is correct
                
                // Custom verification
                harness.AssertFileContains(outputDir, "values.yaml", 
                    "myOperator:", "version: v1.0.0")
                harness.AssertFileContains(outputDir, "scripts/install.sh",
                    "helm install my-operator")
            },
        },
        {
            name:    "missing required measurements",
            recipe:  &recipe.Recipe{},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := harness.RunTest(tt.recipe, tt.wantErr)
            if !tt.wantErr && tt.verify != nil {
                tt.verify(t, result.OutputDir)
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
                            "my-operator": measurement.Str("v1.0.0"),
                        },
                    },
                },
            },
        },
    }
}
```

### Step 8: Test Your Bundler

```shell
# Build CLI with new bundler
make build

# Test bundle generation (automatic registration via init())
./dist/eidos_*/eidos bundle \
    --recipe examples/recipes/gb200-eks-ubuntu-training.yaml \
  --bundlers my-bundler \
  --output ./test-bundles

# Verify bundle structure
tree test-bundles/my-bundler/
```

## Helper Functions

### Internal Package Utilities

The `pkg/bundler/internal` package provides:

**Recipe extraction:**
```go
ExtractK8sImageSubtype(recipe)          // Get K8s image measurements
ExtractK8sRegistrySubtype(recipe)       // Get registry info
ExtractGPUDeviceSubtype(recipe)         // Get GPU device measurements
ExtractOSReleaseSubtype(recipe)         // Get OS release info
BuildBaseConfigMap(recipe, additional)  // Extract configuration strings
```

**Context extraction:**
```go
GetFieldWithContext(subtype, fieldName, defaultValue)  // Get field with context
GetSubtypeContext(subtype)                              // Get subtype-level context
```

**Template helpers:**
```go
GetCurrentTimestamp()        // Current timestamp
GetVersion()                 // Bundler version
GetRecipeVersion(recipe)     // Recipe version
GetConfigValue(map, key, default)  // Get config with fallback
```

## Best Practices

### 1. Use Typed Structs for Templates

**❌ Bad (maps):**
```go
data := map[string]interface{}{
    "version": "1.0.0",
    "enabled": true,
}
```

**✅ Good (typed structs):**
```go
type HelmValues struct {
    Version string
    Enabled bool
}

data := &HelmValues{
    Version: "1.0.0",
    Enabled: true,
}
```

### 2. Pass Structs Directly to Templates

**❌ Bad:**
```go
// Don't convert to map
dataMap := structToMap(helmValues)
b.GenerateFileFromTemplate(ctx, GetTemplate, "values.yaml", path, dataMap, 0644)
```

**✅ Good:**
```go
// Pass struct directly
b.GenerateFileFromTemplate(ctx, GetTemplate, "values.yaml", path, helmValues, 0644)
```

### 3. Access ValueWithContext Fields Properly

In templates:
```yaml
# Access .Value for the actual value
version: {{ .OperatorVersion.Value }}

# Access .Context for explanation (in README)
# Version: {{ .OperatorVersion.Value }}
# Reason: {{ .OperatorVersion.Context }}
```

### 4. Handle Missing Data Gracefully

```go
// Provide defaults
configValue := internal.GetConfigValue(config, "my_setting", "default-value")

// Check existence
if imageSubtype != nil {
    if version, exists := imageSubtype.Data["my-operator"]; exists {
        // Use version
    }
}
```

### 5. Use Template Conditionals

```yaml
{{- if .EnableFeatureX.Value }}
featureX:
  enabled: true
{{- end }}
```

### 6. Add Helpful README Templates

```markdown
# templates/README.md.tmpl
# My Operator Deployment Bundle

Generated: {{ .Timestamp }}

## Configuration

- Operator Version: {{ .Helm.OperatorVersion.Value }}
- Registry: {{ .Helm.DriverRegistry.Value }}

## Installation

```bash
chmod +x scripts/install.sh
./scripts/install.sh
```

## Verification

```bash
kubectl get pods -n my-operator-system
```
```

### 7. Implement Custom Validation

Override `Validate()` if you need custom recipe validation:

```go
func (b *Bundler) Validate(ctx context.Context, r *recipe.Recipe) error {
    // Call base validation first
    if err := b.BaseBundler.Validate(ctx, r); err != nil {
        return err
    }
    
    // Custom validation
    imageSubtype := internal.ExtractK8sImageSubtype(r)
    if imageSubtype == nil {
        return fmt.Errorf("recipe missing K8s image measurements")
    }
    
    if _, exists := imageSubtype.Data["my-operator"]; !exists {
        return fmt.Errorf("recipe missing my-operator version")
    }
    
    return nil
}
```

## Testing Guidelines

### Use TestHarness

```go
harness := internal.NewTestHarness(t, NewBundler())

result := harness.RunTest(recipe, expectError)

// TestHarness automatically verifies:
// - Files exist
// - Checksums valid
// - Directory structure correct

// Add custom assertions
harness.AssertFileContains(outputDir, "values.yaml", "expected content")
harness.AssertFileExecutable(outputDir, "scripts/install.sh")
```

### Test Coverage

Ensure tests cover:
- ✅ Valid recipe with all measurements
- ✅ Missing required measurements
- ✅ Invalid data types
- ✅ Template rendering
- ✅ File generation
- ✅ Checksum calculation
- ✅ Custom validation logic

### Table-Driven Tests

```go
tests := []struct {
    name     string
    recipe   *recipe.Recipe
    wantErr  bool
    verify   func(*testing.T, string)
}{
    {"valid", createValidRecipe(), false, verifyValidBundle},
    {"missing-images", createRecipeWithoutImages(), true, nil},
    {"invalid-version", createRecipeWithInvalidVersion(), true, nil},
}
```

## Common Pitfalls

### 1. Hardcoding Values

**❌ Avoid:**
```go
registry := "nvcr.io/nvidia"  // Hardcoded
```

**✅ Extract from recipe:**
```go
registrySubtype := internal.ExtractK8sRegistrySubtype(r)
registry := registrySubtype.Data["uri"].String()
```

### 2. Ignoring Context Cancellation

**❌ Avoid:**
```go
func (b *Bundler) Make(ctx context.Context, ...) {
    // Long operation without context check
}
```

**✅ Check context:**
```go
select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
    // Continue processing
}
```

### 3. Not Using Internal Helpers

**❌ Reinventing the wheel:**
```go
// Manual extraction
for _, m := range recipe.Measurements {
    if m.Type == "K8s" {
        for _, st := range m.Subtypes {
            if st.Name == "image" {
                // Complex extraction logic...
            }
        }
    }
}
```

**✅ Use helpers:**
```go
imageSubtype := internal.ExtractK8sImageSubtype(recipe)
```

## Advanced Topics

### Custom Template Functions

```go
funcMap := template.FuncMap{
    "upper": strings.ToUpper,
    "quote": func(s string) string { return fmt.Sprintf("%q", s) },
}

tmpl, err := template.New("values.yaml").Funcs(funcMap).Parse(templateContent)
```

### Multi-File Generation

```go
files := []struct{template, output string}{
    {"deployment.yaml", "manifests/deployment.yaml"},
    {"service.yaml", "manifests/service.yaml"},
    {"ingress.yaml", "manifests/ingress.yaml"},
}

for _, f := range files {
    err := b.GenerateFileFromTemplate(ctx, GetTemplate, f.template, 
        filepath.Join(outputDir, f.output), data, 0644)
    if err != nil {
        return nil, err
    }
}
```

### Conditional File Generation

```go
if helmValues.EnableFeatureX.Value == "true" {
    // Generate optional manifest
    err := b.GenerateFileFromTemplate(ctx, GetTemplate, "featurex.yaml",
        filepath.Join(outputDir, "manifests/featurex.yaml"), data, 0644)
}
```

## See Also

- [CONTRIBUTING.md](../../CONTRIBUTING.md) - General contribution guidelines
- [Architecture: Data](data.md) - Recipe data architecture
- [pkg/bundler/gpuoperator](../../pkg/bundler/gpuoperator/) - GPU Operator bundler example
- [pkg/bundler/networkoperator](../../pkg/bundler/networkoperator/) - Network Operator bundler example
- [Testing Guide](../../CONTRIBUTING.md#testing-requirements) - Testing standards
