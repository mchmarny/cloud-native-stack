package bundler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

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
	BundlerTypes []BundleType

	// Sequential indicates whether to run bundlers sequentially.
	// If false, bundlers run in parallel. Default is false (parallel).
	Sequential bool

	// FailFast stops execution on first bundler error.
	// Default is false (continues and collects all errors).
	FailFast bool

	// Config provides bundler-specific configuration.
	Config *BundlerConfig

	// DryRun simulates bundle generation without writing files.
	DryRun bool
}

// Option defines a functional option for configuring DefaultBundler.
type Option func(*DefaultBundler)

// WithBundlerTypes sets the bundler types to execute.
// If not set, all registered bundlers are executed.
func WithBundlerTypes(types []BundleType) Option {
	return func(db *DefaultBundler) {
		db.BundlerTypes = types
	}
}

// WithSequential enables or disables sequential execution of bundlers.
// If true, bundlers run sequentially. If false, bundlers run in parallel.
// Default is false (parallel execution).
func WithSequential(sequential bool) Option {
	return func(db *DefaultBundler) {
		db.Sequential = sequential
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
func WithConfig(config *BundlerConfig) Option {
	return func(db *DefaultBundler) {
		db.Config = config
	}
}

// WithDryRun enables or disables dry-run mode.
// In dry-run mode, no files are written to disk.
// Default is false.
func WithDryRun(dryRun bool) Option {
	return func(db *DefaultBundler) {
		db.DryRun = dryRun
	}
}

// New creates a new DefaultBundler with the given options.
// By default, all registered bundlers are executed unless WithBundlerTypes is specified.
func New(opts ...Option) *DefaultBundler {
	db := &DefaultBundler{
		Config: DefaultBundlerConfig(),
	}
	for _, opt := range opts {
		opt(db)
	}
	return db
}

// Make generates bundles from the given recipe into the specified directory.
// It accepts various options to customize the bundling process.
// Returns a BundleOutput summarizing the results of the bundling operation.
// Errors encountered during the process are returned as well.
func (b *DefaultBundler) Make(ctx context.Context, recipe *recipe.Recipe, dir string) (*BundleOutput, error) {
	start := time.Now()

	// Validate input
	if recipe == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest, "recipe cannot be nil")
	}

	if err := ValidateRecipeStructure(recipe); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest, "recipe validation failed", err)
	}

	if dir == "" {
		dir = "."
	}

	// Create output directory
	if !b.DryRun {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternal,
				fmt.Sprintf("failed to create directory %s", dir), err)
		}
	}

	// Select bundlers to execute
	bundlers := selectBundlers(b.BundlerTypes)
	if len(bundlers) == 0 {
		return nil, errors.New(errors.ErrCodeInvalidRequest, "no bundlers selected")
	}

	slog.Info("starting bundle generation",
		"bundler_count", len(bundlers),
		"output_dir", dir,
		"sequential", b.Sequential,
		"dry_run", b.DryRun,
	)

	// Generate bundles
	var output *BundleOutput
	var err error

	if b.Sequential {
		output, err = b.makeSequential(ctx, recipe, dir, bundlers)
	} else {
		output, err = b.makeParallel(ctx, recipe, dir, bundlers)
	}

	if err != nil {
		return output, err
	}

	output.TotalDuration = time.Since(start)
	output.OutputDir = dir

	slog.Info("bundle generation complete", "summary", output.Summary())

	return output, nil
}

// makeSequential executes bundlers sequentially.
func (b *DefaultBundler) makeSequential(ctx context.Context, recipe *recipe.Recipe, dir string, bundlers map[BundleType]Bundler) (*BundleOutput, error) {
	output := &BundleOutput{
		Results: make([]*BundleResult, 0, len(bundlers)),
		Errors:  make([]BundleError, 0),
	}

	for bundlerType, bundler := range bundlers {
		result, err := b.executeBundler(ctx, bundlerType, bundler, recipe, dir)
		output.Results = append(output.Results, result)

		if result.Success {
			output.TotalSize += result.Size
			output.TotalFiles += len(result.Files)
		}

		if err != nil {
			bundleErr := BundleError{
				BundlerType: bundlerType,
				Error:       err.Error(),
			}
			output.Errors = append(output.Errors, bundleErr)

			slog.Error("bundler failed",
				"bundler_type", bundlerType,
				"error", err,
			)

			if b.FailFast {
				return output, errors.Wrap(errors.ErrCodeInternal,
					fmt.Sprintf("bundler %s failed", bundlerType), err)
			}
		}
	}

	if len(output.Errors) > 0 && b.FailFast {
		return output, errors.New(errors.ErrCodeInternal,
			fmt.Sprintf("%d bundler(s) failed", len(output.Errors)))
	}

	return output, nil
}

// makeParallel executes bundlers concurrently.
func (b *DefaultBundler) makeParallel(ctx context.Context, recipe *recipe.Recipe, dir string, bundlers map[BundleType]Bundler) (*BundleOutput, error) {
	output := &BundleOutput{
		Results: make([]*BundleResult, 0, len(bundlers)),
		Errors:  make([]BundleError, 0),
	}

	g, gctx := errgroup.WithContext(ctx)
	resultChan := make(chan *BundleResult, len(bundlers))
	errorChan := make(chan BundleError, len(bundlers))

	for bundlerType, bundler := range bundlers {
		// Capture loop variables for goroutine
		bundlerType := bundlerType
		bundler := bundler

		g.Go(func() error {
			result, err := b.executeBundler(gctx, bundlerType, bundler, recipe, dir)
			resultChan <- result

			if err != nil {
				errorChan <- BundleError{
					BundlerType: bundlerType,
					Error:       err.Error(),
				}

				if b.FailFast {
					return err
				}
			}
			return nil
		})
	}

	// Wait for all bundlers
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

	return output, nil
}

// executeBundler executes a single bundler and records metrics.
func (b *DefaultBundler) executeBundler(ctx context.Context, bundlerType BundleType, bundler Bundler,
	recipe *recipe.Recipe, dir string) (*BundleResult, error) {

	start := time.Now()
	result := NewBundleResult(bundlerType)

	slog.Debug("executing bundler",
		"bundler_type", bundlerType,
		"output_dir", dir,
	)

	// Configure bundler if it supports configuration
	// NOTE: If bundler stores mutable state, Configure() should use synchronization
	// for thread-safety when DefaultBundler.Make() is called concurrently.
	if cb, ok := bundler.(ConfigurableBundler); ok && b.Config != nil {
		if err := cb.Configure(b.Config); err != nil {
			recordBundleError(bundlerType, "configuration")
			return result, err
		}
	}

	// Validate if bundler supports validation
	if v, ok := bundler.(Validator); ok {
		if err := v.Validate(ctx, recipe); err != nil {
			recordValidationFailure(bundlerType)
			recordBundleError(bundlerType, "validation")
			return result, err
		}
	}

	// Execute bundler
	if !b.DryRun {
		bundlerResult, err := bundler.Make(ctx, recipe, dir)
		if err != nil {
			result.Duration = time.Since(start)
			recordBundleGenerated(bundlerType, false)
			recordBundleDuration(bundlerType, result.Duration.Seconds())
			recordBundleError(bundlerType, "execution")
			return result, err
		}
		result = bundlerResult
	}

	result.Duration = time.Since(start)
	result.MarkSuccess()

	// Record metrics
	recordBundleGenerated(bundlerType, true)
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
func selectBundlers(types []BundleType) map[BundleType]Bundler {
	if len(types) == 0 {
		// Return all registered bundlers
		return defaultRegistry.GetAll()
	}

	// Return only specified bundlers
	selected := make(map[BundleType]Bundler)
	for _, t := range types {
		if b, ok := defaultRegistry.Get(t); ok {
			selected[t] = b
		}
	}
	return selected
}
