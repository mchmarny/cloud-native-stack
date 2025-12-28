/*
Package bundler provides a framework for generating deployment bundles from recipes.

The bundler framework transforms recipe configurations (system measurements and
recommendations) into deployment-ready artifacts including Helm values, Kubernetes
manifests, installation scripts, and documentation.

# Architecture

The package uses these design patterns:

  - Registry Pattern: Thread-safe bundler registry with sync.RWMutex
  - Factory Pattern: DefaultBundler orchestrates bundle generation
  - Functional Options: Configuration via WithBundlerTypes, WithConfig, etc.
  - Builder Pattern: bundle.Result for constructing bundler outputs

# Core Components

  - DefaultBundler: Main orchestrator for bundle generation
  - Registry: Thread-safe storage for bundler implementations
  - bundle.Bundler: Interface that all bundlers must implement
  - bundle.Result: Individual bundler execution result
  - bundle.Output: Aggregated results from all bundlers
  - config.Config: Configuration options for bundlers

# Quick Start

Basic usage with default settings (generates GPU Operator bundle):

	b := bundler.New()
	output, err := b.Make(ctx, recipe, "./bundles")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Generated: %s\n", output.Summary())

Customize with functional options:

	b := bundler.New(
		bundler.WithBundlerTypes([]bundle.Type{bundle.BundleTypeGpuOperator}),
		bundler.WithFailFast(true),
		bundler.WithDryRun(true),
	)

Use custom configuration:

	cfg := config.NewConfig()
	cfg.Namespace = "gpu-operator"
	cfg.IncludeScripts = true

	b := bundler.New(bundler.WithConfig(cfg))

# Bundle Types

Currently supported bundlers:

  - gpu-operator: Generates GPU Operator deployment bundles
  - Helm values.yaml
  - ClusterPolicy manifest
  - Installation/uninstallation scripts
  - README documentation
  - SHA256 checksums

To add custom bundlers, implement bundle.Bundler interface and register:

	reg := bundler.NewRegistry()
	reg.Register("my-bundler", myBundlerImpl)
	b := bundler.New(bundler.WithRegistry(reg))

# Execution Modes

## Parallel Execution (Default)

Bundlers run concurrently for better performance:

	b := bundler.New() // Parallel by default

Benefits:
  - Faster execution with multiple bundlers
  - Efficient resource utilization
  - Context cancellation propagated to all goroutines

## Sequential Execution

Bundlers run one at a time:

	b := bundler.New(bundler.WithSequential(true))

Use when:
  - Debugging bundler behavior
  - Resource constraints
  - Deterministic execution order required

# Error Handling

## Collect All Errors (Default)

All bundlers execute, errors are collected:

	output, err := b.Make(ctx, recipe, dir)
	if output.HasErrors() {
		for _, e := range output.Errors {
			fmt.Printf("Bundler %s failed: %s\n", e.BundlerType, e.Error)
		}
	}
	if err != nil {
		// Error indicates multiple failures or system error
	}

## Fail-Fast Mode

Stop on first error:

	b := bundler.New(bundler.WithFailFast(true))
	output, err := b.Make(ctx, recipe, dir)
	if err != nil {
		// First bundler failure
		return err
	}

# Configuration

Customize bundler behavior with config.Config:

	cfg := config.NewConfig()
	cfg.OutputFormat = "yaml"          // Output format: yaml, json, helm
	cfg.Namespace = "gpu-operator"      // Kubernetes namespace
	cfg.IncludeScripts = true           // Generate install/uninstall scripts
	cfg.IncludeReadme = true            // Generate README documentation
	cfg.IncludeChecksums = true         // Generate SHA256 checksums
	cfg.Compression = false             // Enable tar.gz compression
	cfg.Verbose = false                 // Enable verbose logging

	// Custom labels and annotations
	cfg.CustomLabels["environment"] = "production"
	cfg.CustomAnnotations["owner"] = "platform-team"

	b := bundler.New(bundler.WithConfig(cfg))

# Registry Management

Create and populate custom registry:

	reg := bundler.NewRegistry()
	reg.Register(bundle.BundleTypeGpuOperator, gpuoperator.NewBundler(cfg))
	reg.Register("network-operator", networkoperator.NewBundler(cfg))

	// Use custom registry
	b := bundler.New(bundler.WithRegistry(reg))

Registry operations are thread-safe:

	// List registered bundlers
	types := reg.List()

	// Get specific bundler
	bundler, ok := reg.Get(bundle.BundleTypeGpuOperator)

	// Unregister bundler
	err := reg.Unregister("network-operator")

# Results and Output

## Individual Results

Each bundler returns a bundle.Result:

	result := &bundle.Result{
		Type:     bundle.BundleTypeGpuOperator,
		Files:    []string{"values.yaml", "scripts/install.sh"},
		Duration: 250 * time.Millisecond,
		Size:     4096,
		Success:  true,
	}

## Aggregated Output

DefaultBundler.Make() returns bundle.Output:

	output := &bundle.Output{
		Results:       []*bundle.Result{result1, result2},
		TotalSize:     8192,
		TotalFiles:    5,
		TotalDuration: 500 * time.Millisecond,
		Errors:        []bundle.BundleError{},
		OutputDir:     "./bundles",
	}

	fmt.Println(output.Summary())
	// Output: Generated 5 files (8.0 KB) in 500ms. Success: 2/2 bundlers.

# Thread Safety

Concurrency guarantees:

  - Registry: Thread-safe for concurrent reads/writes (sync.RWMutex)
  - DefaultBundler: Safe for concurrent Make() calls
  - Bundler implementations: Should be stateless or use synchronization

DefaultBundler uses errgroup.WithContext for parallel execution:
  - Goroutine coordination
  - Context cancellation propagation
  - Error collection

# Observability

## Structured Logging

All operations log via slog:

	slog.Info("starting bundle generation",
		"bundler_count", 2,
		"output_dir", "./bundles",
	)

Log levels:
  - Debug: Detailed bundler execution
  - Info: Normal operations and completion
  - Warn: Non-fatal issues
  - Error: Bundler failures

## Prometheus Metrics

Automatically exported metrics:

  - eidos_bundles_generated_total{bundler_type, status}
    Counter of bundle generations by type and status

  - eidos_bundle_duration_seconds{bundler_type}
    Histogram of bundle generation duration

  - eidos_bundle_size_bytes{bundler_type}
    Gauge of generated bundle size

  - eidos_bundle_files_total{bundler_type}
    Gauge of files generated per bundle

  - eidos_bundle_errors_total{bundler_type, error_type}
    Counter of errors during generation

  - eidos_bundle_validation_failures_total{bundler_type}
    Counter of validation failures

Metrics are automatically recorded during Make() execution.

# Creating Custom Bundlers

Implement the bundle.Bundler interface:

	package mybundler

	import (
		"context"
		"github.com/NVIDIA/cloud-native-stack/pkg/bundler/bundle"
		"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
		"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	)

	const BundleTypeMyBundler bundle.Type = "my-bundler"

	type Bundler struct {
		config *config.Config
	}

	func NewBundler(cfg *config.Config) *Bundler {
		if cfg == nil {
			cfg = config.NewConfig()
		}
		return &Bundler{config: cfg}
	}

	func (b *Bundler) Make(ctx context.Context, recipe *recipe.Recipe,
		dir string) (*bundle.Result, error) {

		result := bundle.NewResult(BundleTypeMyBundler)

		// Validate recipe
		if err := recipe.ValidateMeasurementExists(measurement.TypeK8s); err != nil {
			return result, err
		}

		// Generate bundle files
		// ...

		result.AddFile(filepath, size)
		result.MarkSuccess()
		return result, nil
	}

Best practices:
  - Keep bundlers stateless
  - Use config.Config for all configuration
  - Validate recipe early
  - Check context cancellation for long operations
  - Add structured logging
  - Use bundle.Result to track files and errors

# Examples

## Generate All Bundles

	func generateAll(recipe *recipe.Recipe) error {
		b := bundler.New()
		output, err := b.Make(context.Background(), recipe, "./output")
		if err != nil {
			return fmt.Errorf("generation failed: %w", err)
		}
		fmt.Println(output.Summary())
		return nil
	}

## Generate Specific Bundle with Custom Config

	func generateGPUOperator(recipe *recipe.Recipe) error {
		cfg := config.NewConfig()
		cfg.Namespace = "nvidia-gpu-operator"
		cfg.CustomLabels["env"] = "prod"

		b := bundler.New(
			bundler.WithBundlerTypes([]bundle.Type{bundle.BundleTypeGpuOperator}),
			bundler.WithConfig(cfg),
			bundler.WithFailFast(true),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		output, err := b.Make(ctx, recipe, "./gpu-operator-bundle")
		if err != nil {
			return err
		}

		for _, result := range output.Results {
			fmt.Printf("Bundler %s: %d files, %d bytes\n",
				result.Type, len(result.Files), result.Size)
		}

		return nil
	}

## Custom Registry with Multiple Bundlers

	func setupCustomBundlers() *bundler.DefaultBundler {
		cfg := config.NewConfig()
		reg := bundler.NewRegistry()

		// Register multiple bundlers
		reg.Register(bundle.BundleTypeGpuOperator, gpuoperator.NewBundler(cfg))
		reg.Register("network-operator", networkoperator.NewBundler(cfg))
		reg.Register("storage-operator", storageoperator.NewBundler(cfg))

		return bundler.New(
			bundler.WithRegistry(reg),
		)
	}

# Performance Considerations

  - All bundlers execute in parallel for optimal performance
  - Use WithFailFast when early termination is acceptable
  - Context cancellation stops all running bundlers
  - Registry read operations are optimized with RWMutex

# Error Scenarios

Common errors and handling:

	// Recipe validation failed
	if errors.Is(err, errors.ErrCodeInvalidRequest) {
		// Fix recipe structure or measurements
	}

	// Bundler execution failed
	if output.HasErrors() {
		for _, e := range output.Errors {
			// Handle specific bundler failures
		}
	}

	// Context cancelled
	if errors.Is(err, context.Canceled) {
		// User interrupted or timeout reached
	}

	// No bundlers selected
	if errors.Is(err, errors.ErrCodeInvalidRequest) {
		// Check bundler types or registry
	}

See https://github.com/NVIDIA/cloud-native-stack for more information.
*/
package bundler
