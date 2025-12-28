// Package bundler provides functionality to create deployment bundles
// from Cloud Native Stack recipes. It generates complete deployment artifacts
// including Helm values, Kubernetes manifests, installation scripts, and documentation.
//
// # Architecture
//
// The bundler package uses a registry-based architecture where bundlers
// are registered at init-time and retrieved dynamically. Each bundler generates
// deployment artifacts for a specific component (e.g., GPU Operator, Network Operator).
//
// The DefaultBundler orchestrates bundle generation with support for:
//   - Sequential or parallel bundler execution
//   - Fail-fast or error collection modes
//   - Dry-run simulation
//   - Bundler-specific configuration
//
// # Bundler Interface
//
// Bundlers implement the Bundler interface:
//
//	type Bundler interface {
//	    Make(ctx context.Context, recipe *recipe.Recipe, dir string) (*BundleResult, error)
//	}
//
// Optional interfaces:
//
//	ConfigurableBundler - Supports runtime configuration
//	Validator - Validates recipes before bundling
//
// # Usage
//
// Basic usage:
//
//	recipe, _ := recipe.NewBuilder().BuildFromQuery(ctx, query)
//	b := bundler.New(bundler.WithBundlerTypes([]bundler.BundleType{"gpu-operator"}))
//	output, err := b.Make(ctx, recipe, "./output")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(output.Summary())
//
// With all options:
//
//	config := bundler.DefaultBundlerConfig()
//	config.Namespace = "gpu-operator"
//
//	b := bundler.New(
//	    bundler.WithBundlerTypes([]bundler.BundleType{"gpu-operator"}),
//	    bundler.WithSequential(false), // false = parallel (default), true = sequential
//	    bundler.WithFailFast(true),
//	    bundler.WithConfig(config),
//	    bundler.WithDryRun(false),
//	)
//	output, err := b.Make(ctx, recipe, "./output")
//
// # Adding New Bundlers
//
// To add a new bundler:
//
// 1. Create a package under pkg/bundler/yourcomponent/
// 2. Implement the Bundler interface
// 3. Register in init():
//
//	func init() {
//	    bundler.Register("your-component", &YourBundler{})
//	}
//
// 4. Optionally implement ConfigurableBundler and/or Validator
//
// See pkg/bundler/gpuoperator/ for a complete reference implementation.
//
// # Thread Safety
//
// DefaultBundler is safe for concurrent reads - multiple goroutines can call Make()
// concurrently on the same instance. However, bundlers from the registry are shared
// instances. If a bundler implements ConfigurableBundler and stores mutable state,
// it must use synchronization (e.g., sync.RWMutex) to be thread-safe across
// concurrent Make() calls.
//
// Within a single Make() call, bundlers are executed either sequentially or in
// parallel based on the Sequential option (false=parallel by default). Each bundler
// is executed only once per call.
//
// # Bundle Structure
//
// Generated bundles follow this structure:
//
//	output_dir/
//	  gpu-operator/              # One directory per bundler
//	    values.yaml              # Helm chart values
//	    manifests/               # Kubernetes manifests
//	      clusterpolicy.yaml
//	    scripts/                 # Installation scripts
//	      install.sh
//	      uninstall.sh
//	    README.md                # Deployment documentation
//	    checksums.txt            # SHA256 checksums
//
// # Error Handling
//
// Bundlers use structured errors from pkg/errors for consistent error handling.
// The DefaultBundler supports two error modes:
//
//   - FailFast=false (default): Collects all errors and returns BundleOutput with error details
//   - FailFast=true: Stops on first error and returns immediately
//
// Check for errors:
//
//	output, err := b.Make(ctx, recipe, "./output")
//	if err != nil {
//	    // Critical failure or FailFast mode
//	}
//	if output.HasErrors() {
//	    // Some bundlers failed (FailFast=false mode)
//	    for _, e := range output.Errors {
//	        log.Printf("Bundler %s failed: %s", e.BundlerType, e.Error)
//	    }
//	}
//
// # Observability
//
// The bundler package exposes Prometheus metrics:
//
//	bundler_make_duration_seconds      - Bundle generation duration histogram
//	bundler_make_total                 - Total bundles generated counter (by result)
//	bundler_files_generated_total      - Files generated gauge
//	bundler_bytes_generated_total      - Bytes generated gauge
//	bundler_validation_failures_total  - Validation failures counter
//
// Metrics are labeled by bundler_type for per-bundler observability.
//
// # Validation
//
// Recipes are validated before bundling using ValidateRecipeStructure():
//
//	if err := bundler.ValidateRecipeStructure(recipe); err != nil {
//	    // Handle validation error
//	}
//
// Individual bundlers can implement the Validator interface for component-specific validation.
//
// # Configuration
//
// Bundlers can be configured using BundlerConfig:
//
//	config := bundler.DefaultBundlerConfig()
//	config.OutputFormat = "yaml"
//	config.Namespace = "gpu-operator"
//	config.IncludeScripts = true
//	config.IncludeChecksums = true
//	config.HelmRepository = "https://helm.ngc.nvidia.com/nvidia"
//
// # Dry Run Mode
//
// Test bundle generation without writing files:
//
//	b := bundler.New(bundler.WithDryRun(true))
//	output, _ := b.Make(ctx, recipe, "./output")
//	// No files written, but validation and logic are executed
package bundler
