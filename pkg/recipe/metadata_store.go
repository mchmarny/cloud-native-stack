package recipe

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	cnserrors "github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"gopkg.in/yaml.v3"
)

//go:embed data/base.yaml data/*.yaml
var metadataFS embed.FS

var (
	metadataStoreOnce   sync.Once
	cachedMetadataStore *MetadataStore
	cachedMetadataErr   error
)

// MetadataStore holds the base recipe and all overlays.
type MetadataStore struct {
	// Base is the base recipe metadata.
	Base *RecipeMetadata

	// Overlays is a list of overlay recipes indexed by name.
	Overlays map[string]*RecipeMetadata

	// ValuesFiles contains embedded values file contents indexed by filename.
	ValuesFiles map[string][]byte
}

// loadMetadataStore loads and caches the metadata store from embedded data.
func loadMetadataStore(_ context.Context) (*MetadataStore, error) {
	metadataStoreOnce.Do(func() {
		// Record cache miss on first load
		recipeCacheMisses.Inc()

		store := &MetadataStore{
			Overlays:    make(map[string]*RecipeMetadata),
			ValuesFiles: make(map[string][]byte),
		}

		// Load all YAML files from data directory
		err := fs.WalkDir(metadataFS, "data", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			filename := filepath.Base(path)

			// Handle component files (files in the components/ directory)
			if strings.Contains(path, "components/") {
				content, readErr := metadataFS.ReadFile(path)
				if readErr != nil {
					return fmt.Errorf("failed to read component file %s: %w", path, readErr)
				}
				// Store with relative path from data/ directory (e.g., "components/cert-manager/values.yaml")
				relPath := strings.TrimPrefix(path, "data/")
				store.ValuesFiles[relPath] = content
				return nil
			}

			// Skip non-YAML files
			if !strings.HasSuffix(filename, ".yaml") {
				return nil
			}

			// Skip old data-v1.yaml format
			if filename == "data-v1.yaml" {
				return nil
			}

			// Read and parse metadata file
			content, readErr := metadataFS.ReadFile(path)
			if readErr != nil {
				return fmt.Errorf("failed to read %s: %w", path, readErr)
			}

			var metadata RecipeMetadata
			if parseErr := yaml.Unmarshal(content, &metadata); parseErr != nil {
				return fmt.Errorf("failed to parse %s: %w", path, parseErr)
			}

			// Categorize as base or overlay
			if filename == "base.yaml" {
				store.Base = &metadata
			} else {
				store.Overlays[metadata.Metadata.Name] = &metadata
			}

			return nil
		})

		if err != nil {
			cachedMetadataErr = err
			return
		}

		if store.Base == nil {
			cachedMetadataErr = cnserrors.New(cnserrors.ErrCodeInternal, "base.yaml not found")
			return
		}

		// Validate base recipe dependencies
		if err := store.Base.Spec.ValidateDependencies(); err != nil {
			cachedMetadataErr = cnserrors.Wrap(cnserrors.ErrCodeInvalidRequest, "base recipe validation failed", err)
			return
		}

		cachedMetadataStore = store
	})

	// Record cache hit if store was already loaded (not on first load)
	if cachedMetadataStore != nil && cachedMetadataErr == nil {
		recipeCacheHits.Inc()
	}

	if cachedMetadataErr != nil {
		return nil, cachedMetadataErr
	}
	if cachedMetadataStore == nil {
		return nil, cnserrors.New(cnserrors.ErrCodeInternal, "metadata store not initialized")
	}
	return cachedMetadataStore, nil
}

// GetValuesFile returns the content of a values file by filename.
func (s *MetadataStore) GetValuesFile(filename string) ([]byte, error) {
	content, exists := s.ValuesFiles[filename]
	if !exists {
		return nil, cnserrors.New(cnserrors.ErrCodeNotFound, fmt.Sprintf("values file not found: %s", filename))
	}
	return content, nil
}

// GetRecipeByName returns a recipe metadata by name.
// Returns the base recipe if name is "base", otherwise looks up in overlays.
func (s *MetadataStore) GetRecipeByName(name string) (*RecipeMetadata, bool) {
	if name == "" || name == "base" {
		return s.Base, s.Base != nil
	}
	overlay, exists := s.Overlays[name]
	return overlay, exists
}

// resolveInheritanceChain builds the inheritance chain for a recipe.
// Returns recipes in order from root (base) to the target recipe.
// Detects cycles in the inheritance chain.
func (s *MetadataStore) resolveInheritanceChain(recipeName string) ([]*RecipeMetadata, error) {
	// Track visited recipes to detect cycles
	visited := make(map[string]bool)
	var chain []*RecipeMetadata

	currentName := recipeName
	for currentName != "" && currentName != "base" {
		// Check for cycle
		if visited[currentName] {
			return nil, cnserrors.New(cnserrors.ErrCodeInvalidRequest,
				fmt.Sprintf("circular inheritance detected: recipe %q references itself in inheritance chain", currentName))
		}
		visited[currentName] = true

		// Get the recipe
		recipe, exists := s.GetRecipeByName(currentName)
		if !exists {
			return nil, cnserrors.New(cnserrors.ErrCodeNotFound,
				fmt.Sprintf("recipe %q not found (referenced in inheritance chain)", currentName))
		}

		// Prepend to chain (we're walking backwards, will reverse later)
		chain = append([]*RecipeMetadata{recipe}, chain...)

		// Move to parent
		currentName = recipe.Spec.Base
	}

	// Prepend base at the start (root of all inheritance)
	if s.Base != nil {
		chain = append([]*RecipeMetadata{s.Base}, chain...)
	}

	return chain, nil
}

// FindMatchingOverlays finds all overlays that match the given criteria.
// Returns overlays sorted by specificity (least specific first).
func (s *MetadataStore) FindMatchingOverlays(criteria *Criteria) []*RecipeMetadata {
	var matches []*RecipeMetadata

	for _, overlay := range s.Overlays {
		if overlay.Spec.Criteria == nil {
			continue
		}
		if overlay.Spec.Criteria.Matches(criteria) {
			matches = append(matches, overlay)
		}
	}

	// Sort by specificity (least specific first, so more specific overlays are applied later)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Spec.Criteria.Specificity() < matches[j].Spec.Criteria.Specificity()
	})

	return matches
}

// BuildRecipeResult builds a RecipeResult by merging base with matching overlays.
// Each matching overlay is resolved through its inheritance chain before merging.
// This enables multi-level inheritance: base → intermediate → overlay.
func (s *MetadataStore) BuildRecipeResult(ctx context.Context, criteria *Criteria) (*RecipeResult, error) {
	// Check if ctx has been canceled and exit early if so
	select {
	case <-ctx.Done():
		return nil, cnserrors.WrapWithContext(
			cnserrors.ErrCodeTimeout,
			"build recipe result context cancelled during initialization",
			ctx.Err(),
			map[string]any{
				"stage": "initialization",
			},
		)
	default:
	}

	// Find matching overlays (sorted by specificity, least specific first)
	overlays := s.FindMatchingOverlays(criteria)

	// Track all applied recipes (from inheritance chains)
	appliedOverlays := make([]string, 0)

	// Start with the base spec
	mergedSpec := RecipeMetadataSpec{
		Constraints:   make([]Constraint, len(s.Base.Spec.Constraints)),
		ComponentRefs: make([]ComponentRef, len(s.Base.Spec.ComponentRefs)),
	}
	copy(mergedSpec.Constraints, s.Base.Spec.Constraints)
	copy(mergedSpec.ComponentRefs, s.Base.Spec.ComponentRefs)
	appliedOverlays = append(appliedOverlays, "base")

	// For each matching overlay, resolve its inheritance chain and merge
	// We only apply the leaf overlay's chain, not intermediate ones
	// This avoids double-applying recipes that appear in multiple chains
	processedChains := make(map[string]bool) // Track which recipes we've already applied

	for _, overlay := range overlays {
		// Resolve the full inheritance chain for this overlay
		chain, err := s.resolveInheritanceChain(overlay.Metadata.Name)
		if err != nil {
			return nil, cnserrors.WrapWithContext(
				cnserrors.ErrCodeInvalidRequest,
				"failed to resolve inheritance chain",
				err,
				map[string]any{
					"overlay": overlay.Metadata.Name,
				},
			)
		}

		// Apply each recipe in the chain that hasn't been applied yet
		// Skip base (index 0) since we already started with it
		for i := 1; i < len(chain); i++ {
			recipe := chain[i]
			if processedChains[recipe.Metadata.Name] {
				continue // Already applied this recipe
			}
			processedChains[recipe.Metadata.Name] = true
			mergedSpec.Merge(&recipe.Spec)
			appliedOverlays = append(appliedOverlays, recipe.Metadata.Name)
		}
	}

	// Warn if no overlays matched - user is getting base-only configuration
	if len(appliedOverlays) <= 1 { // Only "base" was applied
		slog.Warn("no environment-specific overlays matched, using base configuration only",
			"criteria", criteria.String(),
			"hint", "recipe may not be optimized for your environment")
	}

	// Validate merged dependencies
	if err := mergedSpec.ValidateDependencies(); err != nil {
		return nil, cnserrors.Wrap(cnserrors.ErrCodeInvalidRequest, "merged recipe validation failed", err)
	}

	// Compute deployment order
	deployOrder, err := mergedSpec.TopologicalSort()
	if err != nil {
		return nil, cnserrors.Wrap(cnserrors.ErrCodeInternal, "failed to compute deployment order", err)
	}

	// Build result
	result := &RecipeResult{
		Kind:            "recipeResult",
		APIVersion:      "cns.nvidia.com/v1alpha1",
		Criteria:        criteria,
		Constraints:     mergedSpec.Constraints,
		ComponentRefs:   mergedSpec.ComponentRefs,
		DeploymentOrder: deployOrder,
	}
	result.Metadata.GeneratedAt = time.Now().UTC()
	result.Metadata.AppliedOverlays = appliedOverlays

	return result, nil
}

// BuildRecipeResultWithEvaluator builds a RecipeResult by merging base with matching overlays,
// filtering overlays based on constraint evaluation using the provided evaluator function.
//
// This method extends BuildRecipeResult with constraint-aware filtering:
//   - Each overlay that matches by criteria is tested against its constraints
//   - Overlays with failing constraints are excluded from the merge
//   - Warnings about excluded overlays are included in the result metadata
//
// The evaluator function is called for each constraint in each matching overlay.
// If evaluator is nil, this method behaves identically to BuildRecipeResult.
func (s *MetadataStore) BuildRecipeResultWithEvaluator(ctx context.Context, criteria *Criteria, evaluator ConstraintEvaluatorFunc) (*RecipeResult, error) {
	// If no evaluator provided, use the standard build method
	if evaluator == nil {
		return s.BuildRecipeResult(ctx, criteria)
	}

	// Check if ctx has been canceled
	select {
	case <-ctx.Done():
		return nil, cnserrors.WrapWithContext(
			cnserrors.ErrCodeTimeout,
			"build recipe result context cancelled during initialization",
			ctx.Err(),
			map[string]any{
				"stage": "initialization",
			},
		)
	default:
	}

	// Find matching overlays (sorted by specificity, least specific first)
	overlays := s.FindMatchingOverlays(criteria)

	// Evaluate constraints and filter overlays
	var filteredOverlays []*RecipeMetadata
	var excludedOverlays []string
	var constraintWarnings []ConstraintWarning

	for _, overlay := range overlays {
		slog.Debug("evaluating overlay constraints",
			"overlay", overlay.Metadata.Name,
			"constraint_count", len(overlay.Spec.Constraints))

		passed, warnings := s.evaluateOverlayConstraints(overlay, evaluator)
		if passed {
			filteredOverlays = append(filteredOverlays, overlay)
			slog.Debug("overlay passed all constraints",
				"overlay", overlay.Metadata.Name)
		} else {
			excludedOverlays = append(excludedOverlays, overlay.Metadata.Name)
			constraintWarnings = append(constraintWarnings, warnings...)
			slog.Info("excluding overlay due to constraint failures",
				"overlay", overlay.Metadata.Name,
				"failed_constraints", len(warnings))
		}
	}

	// Track all applied recipes (from inheritance chains)
	appliedOverlays := make([]string, 0)

	// Start with the base spec
	mergedSpec := RecipeMetadataSpec{
		Constraints:   make([]Constraint, len(s.Base.Spec.Constraints)),
		ComponentRefs: make([]ComponentRef, len(s.Base.Spec.ComponentRefs)),
	}
	copy(mergedSpec.Constraints, s.Base.Spec.Constraints)
	copy(mergedSpec.ComponentRefs, s.Base.Spec.ComponentRefs)
	appliedOverlays = append(appliedOverlays, "base")

	// For each filtered overlay, resolve its inheritance chain and merge
	processedChains := make(map[string]bool)

	for _, overlay := range filteredOverlays {
		// Resolve the full inheritance chain for this overlay
		chain, err := s.resolveInheritanceChain(overlay.Metadata.Name)
		if err != nil {
			return nil, cnserrors.WrapWithContext(
				cnserrors.ErrCodeInvalidRequest,
				"failed to resolve inheritance chain",
				err,
				map[string]any{
					"overlay": overlay.Metadata.Name,
				},
			)
		}

		// Apply each recipe in the chain that hasn't been applied yet
		// Skip base (index 0) since we already started with it
		for i := 1; i < len(chain); i++ {
			recipe := chain[i]
			if processedChains[recipe.Metadata.Name] {
				continue
			}
			processedChains[recipe.Metadata.Name] = true
			mergedSpec.Merge(&recipe.Spec)
			appliedOverlays = append(appliedOverlays, recipe.Metadata.Name)
		}
	}

	// Log information about filtered overlays
	if len(excludedOverlays) > 0 {
		slog.Warn("some overlays were excluded due to constraint failures",
			"excluded", excludedOverlays,
			"applied", appliedOverlays,
			"criteria", criteria.String())
	}

	// Warn if no overlays were applied
	if len(appliedOverlays) <= 1 {
		if len(excludedOverlays) > 0 {
			slog.Warn("all matching overlays were excluded due to constraint failures, using base configuration only",
				"excluded_count", len(excludedOverlays),
				"criteria", criteria.String())
		} else {
			slog.Warn("no environment-specific overlays matched, using base configuration only",
				"criteria", criteria.String(),
				"hint", "recipe may not be optimized for your environment")
		}
	}

	// Validate merged dependencies
	if err := mergedSpec.ValidateDependencies(); err != nil {
		return nil, cnserrors.Wrap(cnserrors.ErrCodeInvalidRequest, "merged recipe validation failed", err)
	}

	// Compute deployment order
	deployOrder, err := mergedSpec.TopologicalSort()
	if err != nil {
		return nil, cnserrors.Wrap(cnserrors.ErrCodeInternal, "failed to compute deployment order", err)
	}

	// Build result
	result := &RecipeResult{
		Kind:            "recipeResult",
		APIVersion:      "cns.nvidia.com/v1alpha1",
		Criteria:        criteria,
		Constraints:     mergedSpec.Constraints,
		ComponentRefs:   mergedSpec.ComponentRefs,
		DeploymentOrder: deployOrder,
	}
	result.Metadata.GeneratedAt = time.Now().UTC()
	result.Metadata.AppliedOverlays = appliedOverlays
	result.Metadata.ExcludedOverlays = excludedOverlays
	result.Metadata.ConstraintWarnings = constraintWarnings

	return result, nil
}

// evaluateOverlayConstraints evaluates all constraints in an overlay.
// Returns true if all constraints pass, false otherwise.
// Returns warnings for any constraints that failed or had errors.
func (s *MetadataStore) evaluateOverlayConstraints(overlay *RecipeMetadata, evaluator ConstraintEvaluatorFunc) (bool, []ConstraintWarning) {
	if len(overlay.Spec.Constraints) == 0 {
		// No constraints means the overlay passes
		return true, nil
	}

	var warnings []ConstraintWarning
	allPassed := true

	for _, constraint := range overlay.Spec.Constraints {
		result := evaluator(constraint)

		switch {
		case result.Error != nil:
			// Treat evaluation errors as failures with a warning
			warnings = append(warnings, ConstraintWarning{
				Overlay:    overlay.Metadata.Name,
				Constraint: constraint.Name,
				Expected:   constraint.Value,
				Actual:     result.Actual,
				Reason:     result.Error.Error(),
			})
			allPassed = false
			slog.Debug("constraint evaluation error",
				"overlay", overlay.Metadata.Name,
				"constraint", constraint.Name,
				"error", result.Error)
		case !result.Passed:
			warnings = append(warnings, ConstraintWarning{
				Overlay:    overlay.Metadata.Name,
				Constraint: constraint.Name,
				Expected:   constraint.Value,
				Actual:     result.Actual,
				Reason:     fmt.Sprintf("expected %s, got %s", constraint.Value, result.Actual),
			})
			allPassed = false
			slog.Debug("constraint failed",
				"overlay", overlay.Metadata.Name,
				"constraint", constraint.Name,
				"expected", constraint.Value,
				"actual", result.Actual)
		default:
			slog.Debug("constraint passed",
				"overlay", overlay.Metadata.Name,
				"constraint", constraint.Name,
				"expected", constraint.Value,
				"actual", result.Actual)
		}
	}

	return allPassed, warnings
}
