package bundler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/bundle"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"golang.org/x/sync/errgroup"
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
	BundlerTypes []bundle.Type

	// FailFast stops execution on first bundler error.
	// Default is false (continues and collects all errors).
	FailFast bool

	// Config provides bundler-specific configuration.
	Config *config.Config

	// Registry to retrieve bundlers from.
	Registry *Registry
}

// Option defines a functional option for configuring DefaultBundler.
type Option func(*DefaultBundler)

// WithBundlerTypes sets the bundler types to execute.
// If not set, all registered bundlers are executed.
// Nil or empty slice means all bundlers as well.
func WithBundlerTypes(types []bundle.Type) Option {
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
func WithConfig(config *config.Config) Option {
	return func(db *DefaultBundler) {
		db.Config = config
	}
}

// WithRegistry sets the registry to retrieve bundlers from.
// This overrides the default registry created by New().
func WithRegistry(registry *Registry) Option {
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
//   - Creates a new registry with GPU Operator bundler auto-registered
//   - Executes all registered bundlers (use WithBundlerTypes to filter)
//   - Runs bundlers in parallel
//   - Continues on errors (use WithFailFast to stop on first error)
//   - Uses default configuration (use WithConfig to customize)
//
// Example:
//
//	b := bundler.New(
//		bundler.WithBundlerTypes([]bundle.Type{bundle.BundleTypeGpuOperator}),
//		bundler.WithFailFast(true),
//	)
func New(opts ...Option) *DefaultBundler {
	cfg := config.NewConfig()
	reg := NewRegistry(cfg)

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
// Returns a bundle.Output summarizing the results of the bundling operation.
// Errors encountered during the process are returned as well.
func (b *DefaultBundler) Make(ctx context.Context, recipe *recipe.Recipe, dir string) (*bundle.Output, error) {
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

	slog.Info("starting bundle generation",
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

	slog.Info("bundle generation complete", "summary", output.Summary())

	return output, nil
}

// makeParallel executes bundlers concurrently.
func (b *DefaultBundler) makeParallel(ctx context.Context, recipe *recipe.Recipe, dir string, bundlers map[bundle.Type]bundle.Bundler) (*bundle.Output, error) {
	output := &bundle.Output{
		Results: make([]*bundle.Result, 0, len(bundlers)),
		Errors:  make([]bundle.BundleError, 0),
	}

	g, gctx := errgroup.WithContext(ctx)
	resultChan := make(chan *bundle.Result, len(bundlers))
	errorChan := make(chan bundle.BundleError, len(bundlers))

	for bundlerType, bundler := range bundlers {
		// Capture loop variables for goroutine
		bundlerType := bundlerType
		bundler := bundler

		g.Go(func() error {
			result, err := b.executeBundler(gctx, bundlerType, bundler, recipe, dir)

			// Always send result (even if failed)
			select {
			case resultChan <- result:
			case <-gctx.Done():
				return gctx.Err()
			}

			if err != nil {
				select {
				case errorChan <- bundle.BundleError{
					BundlerType: bundlerType,
					Error:       err.Error(),
				}:
				case <-gctx.Done():
					return gctx.Err()
				}

				if b.FailFast {
					return err
				}
			}
			return nil
		})
	}

	// Wait for all bundlers and close channels
	err := g.Wait()
	close(resultChan)
	close(errorChan)

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

	if err != nil && b.FailFast {
		return output, errors.Wrap(errors.ErrCodeInternal, "bundler execution failed", err)
	}

	// When FailFast is false, collect errors but don't return error
	// This allows clients to inspect output.Errors for failures
	return output, nil
}

// executeBundler executes a single bundler and records metrics.
func (b *DefaultBundler) executeBundler(ctx context.Context, bundlerType bundle.Type, bundler bundle.Bundler,
	recipe *recipe.Recipe, dir string) (*bundle.Result, error) {

	start := time.Now()

	// Check if bundler implements Validate method via reflection
	// We can't use a type assertion with *recipe.Recipe in a local interface,
	// so we use a method value approach
	validateMethod := reflect.ValueOf(bundler).MethodByName("Validate")
	if validateMethod.IsValid() && validateMethod.Type().NumIn() == 2 {
		// Call Validate(ctx, recipe)
		results := validateMethod.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(recipe),
		})
		if len(results) > 0 && !results[0].IsNil() {
			err := results[0].Interface().(error)
			recordValidationFailure(bundlerType)
			return bundle.NewResult(bundlerType), errors.Wrap(errors.ErrCodeInvalidRequest,
				fmt.Sprintf("validation failed for bundler %s", bundlerType), err)
		}
	}

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

	slog.Info("bundler completed",
		"bundler_type", bundlerType,
		"files", len(result.Files),
		"size_bytes", result.Size,
		"duration", result.Duration.Round(time.Millisecond),
	)

	return result, nil
}

// selectBundlers selects which bundlers to execute based on options.
func (b *DefaultBundler) selectBundlers(types []bundle.Type) map[bundle.Type]bundle.Bundler {
	if len(types) == 0 {
		return b.Registry.GetAll()
	}

	// Return only specified bundlers
	selected := make(map[bundle.Type]bundle.Bundler)
	for _, t := range types {
		if b, ok := b.Registry.Get(t); ok {
			selected[t] = b
		}
	}
	return selected
}
