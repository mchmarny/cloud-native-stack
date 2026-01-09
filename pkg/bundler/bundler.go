package bundler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/registry"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"

	// Import bundler packages for auto-registration via init()
	_ "github.com/NVIDIA/cloud-native-stack/pkg/bundler/gpuoperator"
	_ "github.com/NVIDIA/cloud-native-stack/pkg/bundler/networkoperator"
	_ "github.com/NVIDIA/cloud-native-stack/pkg/bundler/nvsentinel"
	_ "github.com/NVIDIA/cloud-native-stack/pkg/bundler/skyhook"
)

// DefaultBundler provides default options for bundling operations.
//
// Thread-safety: DefaultBundler is safe for concurrent reads (multiple goroutines
// calling Make() concurrently). However, bundlers retrieved from the registry may
// be shared instances. If ConfigurableBundler.Configure() modifies bundler state,
// concurrent Make() calls may have race conditions. Ensure bundlers are either
// stateless or use synchronization for shared mutable state.
type DefaultBundler struct {
	// BundlerTypes specifies which bundlers to execute.
	// If empty, all registered bundlers are executed.
	BundlerTypes []types.BundleType

	// FailFast stops execution on first bundler error.
	// Default is false (continues and collects all errors).
	FailFast bool

	// Config provides bundler-specific configuration.
	Config *config.Config

	// Registry to retrieve bundlers from.
	Registry *registry.Registry
}

// Option defines a functional option for configuring DefaultBundler.
type Option func(*DefaultBundler)

// WithBundlerTypes sets the bundler types to execute.
// If not set, all registered bundlers are executed.
// Nil or empty slice means all bundlers as well.
func WithBundlerTypes(types []types.BundleType) Option {
	return func(db *DefaultBundler) {
		if len(types) > 0 {
			db.BundlerTypes = types
		}
	}
}

// WithFailFast enables or disables fail-fast behavior.
// If enabled, bundling stops on the first error encountered.
// Default is false.
func WithFailFast(failFast bool) Option {
	return func(db *DefaultBundler) {
		db.FailFast = failFast
	}
}

// WithConfig sets the bundler configuration.
// If nil, default configuration is used.
// Note: This only updates the Config, it does NOT recreate the registry.
// If you need to update both config and registry, call WithConfig before WithRegistry.
func WithConfig(config *config.Config) Option {
	return func(db *DefaultBundler) {
		db.Config = config
	}
}

// WithRegistry sets the registry to retrieve bundlers from.
// This overrides the default registry created by New().
func WithRegistry(registry *registry.Registry) Option {
	return func(db *DefaultBundler) {
		if registry != nil {
			db.Registry = registry
		}
	}
}

// New creates a new DefaultBundler with the given options.
// If no options are provided, default settings are used.
//
// Default behavior:
//   - Creates a registry populated with all globally registered bundlers
//   - Executes all registered bundlers (use WithBundlerTypes to filter)
//   - Runs bundlers in parallel
//   - Continues on errors (use WithFailFast to stop on first error)
//   - Uses default configuration (use WithConfig to customize)
//
// Example:
//
//	b := bundler.New(
//		bundler.WithBundlerTypes([]types.BundleType{types.BundleTypeGpuOperator}),
//		bundler.WithFailFast(true),
//	)
func New(opts ...Option) *DefaultBundler {
	cfg := config.NewConfig()

	// Create registry populated with all globally registered bundlers
	// Bundlers register themselves via init() in their packages
	reg := registry.NewFromGlobal(cfg)

	// Create DefaultBundler with defaults
	db := &DefaultBundler{
		Config:   cfg,
		Registry: reg,
	}

	// Apply options
	for _, opt := range opts {
		opt(db)
	}

	return db
}

// Make generates bundles from the given recipe into the specified directory.
// It accepts various options to customize the bundling process.
// Returns a result.Output summarizing the results of the bundling operation.
// Errors encountered during the process are returned as well.
func (b *DefaultBundler) Make(ctx context.Context, recipe *recipe.Recipe, dir string) (*result.Output, error) {
	start := time.Now()

	// Validate input
	if recipe == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest, "recipe cannot be nil")
	}

	if err := recipe.ValidateStructure(); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest, "recipe validation failed", err)
	}

	// Validate configuration
	if b.Config != nil {
		if err := b.Config.Validate(); err != nil {
			return nil, errors.Wrap(errors.ErrCodeInvalidRequest,
				"invalid configuration", err)
		}
	}

	if dir == "" {
		dir = "."
	}

	// Create output directory if it doesn't exist
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternal,
				"failed to create output directory", err)
		}
	}

	// Select bundlers to execute
	bundlers := b.selectBundlers(b.BundlerTypes)
	if len(bundlers) == 0 {
		return nil, errors.New(errors.ErrCodeInvalidRequest, "no bundlers selected")
	}

	slog.Debug("starting bundle generation",
		"bundler_count", len(bundlers),
		"output_dir", dir,
	)

	// Generate bundles
	output, err := b.makeParallel(ctx, recipe, dir, bundlers)
	if err != nil {
		return output, err
	}

	output.TotalDuration = time.Since(start)
	output.OutputDir = dir

	slog.Debug("bundle generation complete", "summary", output.Summary())

	return output, nil
}

// makeParallel executes bundlers concurrently using buffered channels
// and WaitGroup to prevent deadlocks.
func (b *DefaultBundler) makeParallel(ctx context.Context, recipe *recipe.Recipe, dir string, bundlers map[types.BundleType]registry.Bundler) (*result.Output, error) {
	output := &result.Output{
		Results: make([]*result.Result, 0, len(bundlers)),
		Errors:  make([]result.BundleError, 0),
	}

	// Use buffered channels to prevent blocking on writes
	resultChan := make(chan *result.Result, len(bundlers))
	errorChan := make(chan result.BundleError, len(bundlers))

	var wg sync.WaitGroup
	wg.Add(len(bundlers))

	// Track if we should stop early due to FailFast
	var firstError error
	var firstErrorMu sync.Mutex

	for bundlerType, bundler := range bundlers {
		// Capture loop variables for goroutine
		bundlerType := bundlerType
		bundler := bundler

		go func() {
			defer wg.Done()

			// Check if we should skip due to FailFast
			if b.FailFast {
				firstErrorMu.Lock()
				hasError := firstError != nil
				firstErrorMu.Unlock()
				if hasError {
					return
				}
			}

			res, err := b.executeBundler(ctx, bundlerType, bundler, recipe, dir)

			// Non-blocking writes to buffered channels
			resultChan <- res

			if err != nil {
				errorChan <- result.BundleError{
					BundlerType: bundlerType,
					Error:       err.Error(),
				}

				if b.FailFast {
					firstErrorMu.Lock()
					if firstError == nil {
						firstError = err
					}
					firstErrorMu.Unlock()
				}
			}
		}()
	}

	// Close channels after all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Collect results
	for result := range resultChan {
		output.Results = append(output.Results, result)
		if result.Success {
			output.TotalSize += result.Size
			output.TotalFiles += len(result.Files)
		}
	}

	// Collect errors
	for bundleErr := range errorChan {
		output.Errors = append(output.Errors, bundleErr)
	}

	// Return first error if FailFast is enabled
	if b.FailFast && firstError != nil {
		return output, errors.Wrap(errors.ErrCodeInternal, "bundler execution failed", firstError)
	}

	// When FailFast is false, collect errors but don't return error
	// This allows clients to inspect output.Errors for failures
	return output, nil
}

// executeBundler executes a single bundler and records metrics.
func (b *DefaultBundler) executeBundler(ctx context.Context, bundlerType types.BundleType, bundler registry.Bundler,
	recipe *recipe.Recipe, dir string) (*result.Result, error) {

	start := time.Now()

	// Check if bundler implements ValidatableBundler interface
	// Type assertion is type-safe and 10-100x faster than reflection
	if validator, ok := bundler.(registry.ValidatableBundler); ok {
		if err := validator.Validate(ctx, recipe); err != nil {
			recordValidationFailure(bundlerType)
			return result.New(bundlerType), errors.Wrap(errors.ErrCodeInvalidRequest,
				fmt.Sprintf("validation failed for bundler %s", bundlerType), err)
		}
	}

	// Add context metadata to logging
	slog.Debug("executing bundler",
		"bundler_type", bundlerType,
		"output_dir", dir,
	)

	// Execute the bundler
	result, err := bundler.Make(ctx, recipe, dir)
	if err != nil {
		recordBundleGenerated(bundlerType, false)
		recordBundleError(bundlerType, "execution_error")
		return result, fmt.Errorf("bundler %s failed: %w", bundlerType, err)
	}

	result.Duration = time.Since(start)

	// Record metrics
	recordBundleGenerated(bundlerType, result.Success)
	recordBundleDuration(bundlerType, result.Duration.Seconds())
	recordBundleSize(bundlerType, result.Size)
	recordBundleFiles(bundlerType, len(result.Files))

	slog.Debug("bundler completed",
		"bundler_type", bundlerType,
		"files", len(result.Files),
		"size_bytes", result.Size,
		"duration", result.Duration.Round(time.Millisecond),
	)

	return result, nil
}

// selectBundlers selects which bundlers to execute based on options.
func (b *DefaultBundler) selectBundlers(bundleType []types.BundleType) map[types.BundleType]registry.Bundler {
	if len(bundleType) == 0 {
		return b.Registry.GetAll()
	}

	// Return only specified bundlers
	selected := make(map[types.BundleType]registry.Bundler)
	for _, t := range bundleType {
		if b, ok := b.Registry.Get(t); ok {
			selected[t] = b
		}
	}
	return selected
}
