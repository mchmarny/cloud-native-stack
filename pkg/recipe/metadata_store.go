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

	// Start with a copy of the base spec
	mergedSpec := RecipeMetadataSpec{
		Constraints:   make([]Constraint, len(s.Base.Spec.Constraints)),
		ComponentRefs: make([]ComponentRef, len(s.Base.Spec.ComponentRefs)),
	}
	copy(mergedSpec.Constraints, s.Base.Spec.Constraints)
	copy(mergedSpec.ComponentRefs, s.Base.Spec.ComponentRefs)

	// Find and apply matching overlays
	overlays := s.FindMatchingOverlays(criteria)
	appliedOverlays := make([]string, 0, len(overlays))

	for _, overlay := range overlays {
		mergedSpec.Merge(&overlay.Spec)
		appliedOverlays = append(appliedOverlays, overlay.Metadata.Name)
	}

	// Warn if no overlays matched - user is getting base-only configuration
	if len(appliedOverlays) == 0 {
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
