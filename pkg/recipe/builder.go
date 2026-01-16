package recipe

import (
	"context"
	"time"

	cnserrors "github.com/NVIDIA/cloud-native-stack/pkg/errors"
)

// ConstraintEvalResult represents the result of evaluating a single constraint.
// This mirrors the result from pkg/validator to avoid circular imports.
type ConstraintEvalResult struct {
	// Passed indicates if the constraint was satisfied.
	Passed bool

	// Actual is the actual value extracted from the snapshot.
	Actual string

	// Error contains the error if evaluation failed (e.g., value not found).
	Error error
}

// ConstraintEvaluatorFunc is a function type for evaluating constraints.
// It takes a constraint and returns the evaluation result.
// This function type allows the recipe package to use constraint evaluation
// from the validator package without creating a circular dependency.
type ConstraintEvaluatorFunc func(constraint Constraint) ConstraintEvalResult

// Option is a functional option for configuring Builder instances.
type Option func(*Builder)

// WithVersion returns an Option that sets the Builder version string.
// The version is included in recipe metadata for tracking purposes.
func WithVersion(version string) Option {
	return func(b *Builder) {
		b.Version = version
	}
}

// NewBuilder creates a new Builder instance with the provided functional options.
func NewBuilder(opts ...Option) *Builder {
	b := &Builder{}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Builder constructs RecipeResult payloads based on Criteria specifications.
// It loads recipe metadata, applies matching overlays, and generates
// tailored configuration recipes.
type Builder struct {
	Version string
}

// BuildFromCriteria creates a RecipeResult payload for the provided criteria.
// It loads the metadata store, applies matching overlays, and returns
// a RecipeResult with merged components and computed deployment order.
func (b *Builder) BuildFromCriteria(ctx context.Context, c *Criteria) (*RecipeResult, error) {
	if c == nil {
		return nil, cnserrors.New(cnserrors.ErrCodeInvalidRequest, "criteria cannot be nil")
	}

	// Enforce timeout budget: 25s for building, leaving 5s buffer for handler response
	// This prevents handler deadline from being reached before we can respond
	buildCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	// Check context before expensive operations
	select {
	case <-buildCtx.Done():
		return nil, cnserrors.WrapWithContext(
			cnserrors.ErrCodeTimeout,
			"recipe build context cancelled during initialization",
			buildCtx.Err(),
			map[string]interface{}{
				"stage": "initialization",
			},
		)
	default:
	}

	// Track overall build duration
	start := time.Now()
	defer func() {
		recipeBuiltDuration.Observe(time.Since(start).Seconds())
	}()

	store, err := loadMetadataStore(buildCtx)
	if err != nil {
		return nil, cnserrors.WrapWithContext(
			cnserrors.ErrCodeInternal,
			"failed to load metadata store",
			err,
			map[string]interface{}{
				"stage": "metadata_load",
			},
		)
	}

	result, err := store.BuildRecipeResult(ctx, c)
	if err != nil {
		return nil, err
	}

	// Set recipe version from builder configuration
	if b.Version != "" {
		result.Metadata.Version = b.Version
	}

	return result, nil
}

// BuildFromCriteriaWithEvaluator creates a RecipeResult payload for the provided criteria,
// filtering overlays based on constraint evaluation against snapshot data.
//
// When an evaluator function is provided:
//   - Overlays that match by criteria but fail constraint evaluation are excluded
//   - Constraint warnings are included in the result metadata for visibility
//   - Only overlays whose constraints pass (or have no constraints) are merged
//
// The evaluator function is typically created by wrapping validator.EvaluateConstraint
// with the snapshot data.
func (b *Builder) BuildFromCriteriaWithEvaluator(ctx context.Context, c *Criteria, evaluator ConstraintEvaluatorFunc) (*RecipeResult, error) {
	if c == nil {
		return nil, cnserrors.New(cnserrors.ErrCodeInvalidRequest, "criteria cannot be nil")
	}

	// Enforce timeout budget: 25s for building, leaving 5s buffer for handler response
	buildCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	// Check context before expensive operations
	select {
	case <-buildCtx.Done():
		return nil, cnserrors.WrapWithContext(
			cnserrors.ErrCodeTimeout,
			"recipe build context cancelled during initialization",
			buildCtx.Err(),
			map[string]interface{}{
				"stage": "initialization",
			},
		)
	default:
	}

	// Track overall build duration
	start := time.Now()
	defer func() {
		recipeBuiltDuration.Observe(time.Since(start).Seconds())
	}()

	store, err := loadMetadataStore(buildCtx)
	if err != nil {
		return nil, cnserrors.WrapWithContext(
			cnserrors.ErrCodeInternal,
			"failed to load metadata store",
			err,
			map[string]interface{}{
				"stage": "metadata_load",
			},
		)
	}

	result, err := store.BuildRecipeResultWithEvaluator(ctx, c, evaluator)
	if err != nil {
		return nil, err
	}

	// Set recipe version from builder configuration
	if b.Version != "" {
		result.Metadata.Version = b.Version
	}

	return result, nil
}
