/*
Package bundler provides orchestration for generating deployment bundles from recipes.

The bundler package coordinates parallel execution of component implementations to transform
recipe configurations (system measurements and recommendations) into deployment-ready artifacts
including Helm values, Kubernetes manifests, installation scripts, and documentation.

# Architecture

The package uses these design patterns:

  - Registry Pattern: Thread-safe component registry with sync.RWMutex
  - Factory Pattern: DefaultBundler orchestrates parallel bundle generation
  - Functional Options: Configuration via WithBundlerTypes, WithConfig, etc.
  - Builder Pattern: result.Result for constructing bundler outputs

Component implementations are in pkg/component, which provides:

  - BaseBundler: Helper that reduces boilerplate in component implementations
  - internal: Shared utilities (template rendering, file writing, subtype extraction)
  - Component packages: gpuoperator, networkoperator, certmanager, nvsentinel, skyhook, dradriver

# Core Components

  - DefaultBundler: Main orchestrator for parallel bundle generation
  - Registry: Thread-safe storage for component implementations
  - types.Bundler: Interface that all components must implement
  - result.Result: Individual component execution result
  - result.Output: Aggregated results from all components
  - config.Config: Configuration options for components

# Quick Start

Basic usage with default settings (generates all registered components):

	import (
		_ "github.com/NVIDIA/cloud-native-stack/pkg/component/gpuoperator"  // Auto-registers
	)

	b, err := bundler.New()
	if err != nil {
		log.Fatal(err)
	}
	output, err := b.Make(ctx, recipe, "./bundles")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Generated: %s\n", output.Summary())

Customize with functional options:

	b, err := bundler.New(
		bundler.WithBundlerTypes([]types.BundleType{types.BundleTypeGpuOperator}),
		bundler.WithFailFast(true),
	)
	if err != nil {
		log.Fatal(err)
	}

Use custom configuration:

	cfg := config.NewConfig(
		config.WithNamespace("gpu-operator"),
		config.WithIncludeScripts(true),
	)

	b, err := bundler.New(bundler.WithConfig(cfg))
	if err != nil {
		log.Fatal(err)
	}

# Component Implementation

Component implementations are in pkg/component. See pkg/component/doc.go and
pkg/component/gpuoperator for complete examples of implementing new components.

Components use BaseBundler from pkg/component/internal to reduce boilerplate (~75% less code).
Self-registration via init() enables automatic discovery without manual registration.

# Bundle Types

Currently supported bundlers:

  - gpu-operator: Generates GPU Operator deployment bundles
  - Helm values.yaml
  - ClusterPolicy manifest
  - Installation/uninstallation scripts
  - README documentation
  - SHA256 checksums
  - network-operator: Generates Network Operator deployment bundles
  - Helm values.yaml
  - NicClusterPolicy manifest
  - Installation/uninstallation scripts
  - README documentation
  - SHA256 checksums
  - dra-driver: Generates NVIDIA DRA Driver deployment bundles
  - Helm values.yaml with DRA configuration
  - Installation/uninstallation scripts
  - README documentation
  - SHA256 checksums

# Execution Modes

## Parallel Execution (Default)

Bundlers run concurrently for better performance:

	b, err := bundler.New() // Parallel by default
	if err != nil {
		log.Fatal(err)
	}

Benefits:
  - Faster execution with multiple bundlers
  - Efficient resource utilization
  - Context cancellation propagated to all goroutines

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

	b, err := bundler.New(bundler.WithFailFast(true))
	if err != nil {
		return err
	}
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

	b, err := bundler.New(bundler.WithConfig(cfg))
	if err != nil {
		return err
	}

# Registry Management

Create and populate custom registry:

	reg := bundler.NewRegistry()
	reg.Register(types.BundleTypeGpuOperator, gpuoperator.NewBundler(cfg))
	reg.Register("network-operator", networkoperator.NewBundler(cfg))

	// Use custom registry
	b, err := bundler.New(bundler.WithRegistry(reg))
	if err != nil {
		return err
	}

Registry operations are thread-safe:

	// List registered bundlers
	types := reg.List()

	// Get specific bundler
	bundler, ok := reg.Get(types.BundleTypeGpuOperator)

	// Unregister bundler
	err := reg.Unregister("network-operator")

# Results and Output

## Individual Results

Each bundler returns a result.Result:

	result := &result.Result{
		Type:     types.BundleTypeGpuOperator,
		Files:    []string{"values.yaml", "scripts/install.sh"},
		Duration: 250 * time.Millisecond,
		Size:     4096,
		Success:  true,
	}

## Aggregated Output

DefaultBundler.Make() returns result.Output:

	output := &result.Output{
		Results:       []*result.Result{result1, result2},
		TotalSize:     8192,
		TotalFiles:    5,
		TotalDuration: 500 * time.Millisecond,
		Errors:        []result.BundleError{},
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

	slog.Debug("starting bundle generation",
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

  - cns_bundles_generated_total{bundler_type, status}
    Counter of bundle generations by type and status

  - cns_bundle_duration_seconds{bundler_type}
    Histogram of bundle generation duration

  - cns_bundle_size_bytes{bundler_type}
    Gauge of generated bundle size

  - cns_bundle_files_total{bundler_type}
    Gauge of files generated per bundle

  - cns_bundle_errors_total{bundler_type, error_type}
    Counter of errors during generation

  - cns_bundle_validation_failures_total{bundler_type}
    Counter of validation failures

Metrics are automatically recorded during Make() execution.

# Creating Custom Bundlers

Implement the bundle.Bundler interface:

	package mybundler

	import (
		"context"
		"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
		"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
		"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	)

	const BundleTypeMyBundler types.BundleType = "my-bundler"

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
		dir string) (*result.Result, error) {

		result := result.New(BundleTypeMyBundler)

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
  - Use result.Result to track files and errors

# Examples

## Generate All Bundles

	func generateAll(recipe *recipe.Recipe) error {
		b, err := bundler.New()
		if err != nil {
			return fmt.Errorf("failed to create bundler: %w", err)
		}
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

		b, err := bundler.New(
			bundler.WithBundlerTypes([]types.BundleType{types.BundleTypeGpuOperator}),
			bundler.WithConfig(cfg),
			bundler.WithFailFast(true),
		)
		if err != nil {
			return err
		}

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

	func setupCustomBundlers() (*bundler.DefaultBundler, error) {
		cfg := config.NewConfig()
		reg := bundler.NewRegistry()

		// Register multiple bundlers
		reg.Register(types.BundleTypeGpuOperator, gpuoperator.NewBundler(cfg))
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

	// Context canceled
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
