/*
Package bundler provides orchestration for generating deployment bundles from recipes.

The bundler package coordinates parallel execution of component implementations to transform
recipe configurations into deployment-ready artifacts (Helm values, manifests, scripts).

# Architecture

  - Registry Pattern: Thread-safe component registry
  - DefaultBundler: Orchestrates parallel bundle generation
  - Functional Options: Configuration via WithBundlerTypes, WithConfig, etc.
  - result.Result: Individual component execution result
  - result.Output: Aggregated results from all components

Component implementations are in pkg/component (gpuoperator, networkoperator, etc.).

# Quick Start

	import _ "github.com/NVIDIA/cloud-native-stack/pkg/component/gpuoperator"  // Auto-registers

	b, err := bundler.New()
	output, err := b.Make(ctx, recipe, "./bundles")
	fmt.Printf("Generated: %s\n", output.Summary())

With options:

	b, err := bundler.New(
		bundler.WithBundlerTypes([]types.BundleType{types.BundleTypeGpuOperator}),
		bundler.WithFailFast(true),
	)

# Supported Bundlers

  - gpu-operator: GPU Operator (values.yaml, ClusterPolicy, scripts)
  - network-operator: Network Operator (values.yaml, NicClusterPolicy, scripts)
  - nvidia-dra-driver-gpu: NVIDIA DRA Driver (values.yaml, scripts)
  - cert-manager: Certificate Manager (values.yaml, scripts)
  - nvsentinel: NVSentinel (values.yaml, scripts)
  - skyhook-operator: Skyhook node optimization (values.yaml, Skyhook CR, scripts)
  - prometheus: Prometheus monitoring (values.yaml, scripts)

# Execution

Bundlers run concurrently by default. Use WithFailFast(true) to stop on first error.

# Configuration

	cfg := config.NewConfig(
	    config.WithDeployer(config.DeployerHelm),
	    config.WithIncludeReadme(true),
	)
	b, err := bundler.New(bundler.WithConfig(cfg))

# Error Handling

	output, err := b.Make(ctx, recipe, dir)
	if output.HasErrors() {
		for _, e := range output.Errors {
			fmt.Printf("Bundler %s failed: %s\n", e.BundlerType, e.Error)
		}
	}

# Thread Safety

  - Registry: Thread-safe for concurrent reads/writes
  - DefaultBundler: Safe for concurrent Make() calls
  - Bundler implementations: Should be stateless

# Creating Custom Bundlers

Implement the types.Bundler interface:

	type Bundler struct {
		config *config.Config
	}

	func (b *Bundler) Make(ctx context.Context, recipe *recipe.Recipe,
		dir string) (*result.Result, error) {
		result := result.New(BundleTypeMyBundler)
		// Generate bundle files...
		result.AddFile(filepath, size)
		result.MarkSuccess()
		return result, nil
	}

See pkg/component/gpuoperator for a complete example.

See https://github.com/NVIDIA/cloud-native-stack for more information.
*/
package bundler
