/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/

package recipe

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	"gopkg.in/yaml.v3"
)

// testMetadataFS embeds all recipe data files for testing.
// This uses a separate embed directive to include component values files.
//
//go:embed data/base.yaml data/*.yaml data/components/**/*.yaml
var testMetadataFS embed.FS

// validMeasurementTypes are the valid top-level measurement types for constraints.
var validMeasurementTypes = map[string]bool{
	"K8s":     true,
	"OS":      true,
	"GPU":     true,
	"SystemD": true,
}

// validConstraintOperators are the supported constraint operators.
var validConstraintOperators = []string{">=", "<=", ">", "<", "==", "!="}

// baseYAMLFile is the base recipe filename.
const baseYAMLFile = "base.yaml"

// ============================================================================
// Schema Conformance Tests
// ============================================================================

// TestAllMetadataFilesParseCorrectly verifies that all YAML files in data/
// parse into valid RecipeMetadata structures.
func TestAllMetadataFilesParseCorrectly(t *testing.T) {
	files := collectMetadataFiles(t)
	if len(files) == 0 {
		t.Fatal("no metadata files found")
	}

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := testMetadataFS.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			var metadata RecipeMetadata
			if err := yaml.Unmarshal(content, &metadata); err != nil {
				t.Errorf("failed to parse %s: %v", path, err)
			}
		})
	}
}

// TestAllMetadataFilesHaveRequiredFields verifies that all metadata files
// contain the required fields: kind, apiVersion, metadata.name.
func TestAllMetadataFilesHaveRequiredFields(t *testing.T) {
	files := collectMetadataFiles(t)

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := testMetadataFS.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			var metadata RecipeMetadata
			if err := yaml.Unmarshal(content, &metadata); err != nil {
				t.Fatalf("failed to parse %s: %v", path, err)
			}

			// Check required fields
			if metadata.Kind == "" {
				t.Error("missing required field: kind")
			}
			if metadata.APIVersion == "" {
				t.Error("missing required field: apiVersion")
			}
			if metadata.Metadata.Name == "" {
				t.Error("missing required field: metadata.name")
			}

			// Validate kind and apiVersion values
			if metadata.Kind != "recipeMetadata" {
				t.Errorf("invalid kind: got %q, want %q", metadata.Kind, "recipeMetadata")
			}
			if metadata.APIVersion != "cns.nvidia.com/v1alpha1" {
				t.Errorf("invalid apiVersion: got %q, want %q", metadata.APIVersion, "cns.nvidia.com/v1alpha1")
			}
		})
	}
}

// ============================================================================
// Criteria Validation Tests
// ============================================================================

// TestAllOverlayCriteriaUseValidEnums verifies that all overlay files use
// only valid enum values for criteria fields (service, accelerator, os, intent).
func TestAllOverlayCriteriaUseValidEnums(t *testing.T) {
	files := collectMetadataFiles(t)

	for _, path := range files {
		filename := filepath.Base(path)
		// Skip base.yaml - it doesn't have criteria
		if filename == baseYAMLFile {
			continue
		}

		t.Run(filename, func(t *testing.T) {
			content, err := testMetadataFS.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			var metadata RecipeMetadata
			if err := yaml.Unmarshal(content, &metadata); err != nil {
				t.Fatalf("failed to parse %s: %v", path, err)
			}

			criteria := metadata.Spec.Criteria
			if criteria == nil {
				t.Error("overlay missing criteria field")
				return
			}

			// Validate service type
			if criteria.Service != "" && criteria.Service != CriteriaServiceAny {
				if _, err := ParseCriteriaServiceType(string(criteria.Service)); err != nil {
					t.Errorf("invalid service type %q: %v", criteria.Service, err)
				}
			}

			// Validate accelerator type
			if criteria.Accelerator != "" && criteria.Accelerator != CriteriaAcceleratorAny {
				if _, err := ParseCriteriaAcceleratorType(string(criteria.Accelerator)); err != nil {
					t.Errorf("invalid accelerator type %q: %v", criteria.Accelerator, err)
				}
			}

			// Validate intent type
			if criteria.Intent != "" && criteria.Intent != CriteriaIntentAny {
				if _, err := ParseCriteriaIntentType(string(criteria.Intent)); err != nil {
					t.Errorf("invalid intent type %q: %v", criteria.Intent, err)
				}
			}

			// Validate OS type
			if criteria.OS != "" && criteria.OS != CriteriaOSAny {
				if _, err := ParseCriteriaOSType(string(criteria.OS)); err != nil {
					t.Errorf("invalid OS type %q: %v", criteria.OS, err)
				}
			}
		})
	}
}

// ============================================================================
// Reference Validation Tests
// ============================================================================

// TestAllValuesFileReferencesExist verifies that all valuesFile references
// in componentRefs point to existing files in the data/components/ directory.
func TestAllValuesFileReferencesExist(t *testing.T) {
	files := collectMetadataFiles(t)

	// Build set of available values files
	availableFiles := collectValuesFiles(t)

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := testMetadataFS.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			var metadata RecipeMetadata
			if err := yaml.Unmarshal(content, &metadata); err != nil {
				t.Fatalf("failed to parse %s: %v", path, err)
			}

			for _, comp := range metadata.Spec.ComponentRefs {
				if comp.ValuesFile == "" {
					continue
				}

				if !availableFiles[comp.ValuesFile] {
					t.Errorf("componentRef %q references non-existent valuesFile: %q", comp.Name, comp.ValuesFile)
					t.Logf("available values files: %v", getKeys(availableFiles))
				}
			}
		})
	}
}

// TestAllDependencyReferencesExist verifies that all dependencyRefs
// reference components that are defined in the same file or base.yaml.
func TestAllDependencyReferencesExist(t *testing.T) {
	// Load base components first
	baseContent, err := testMetadataFS.ReadFile("data/" + baseYAMLFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", baseYAMLFile, err)
	}

	var baseMetadata RecipeMetadata
	if err := yaml.Unmarshal(baseContent, &baseMetadata); err != nil {
		t.Fatalf("failed to parse %s: %v", baseYAMLFile, err)
	}

	baseComponents := make(map[string]bool)
	for _, comp := range baseMetadata.Spec.ComponentRefs {
		baseComponents[comp.Name] = true
	}

	files := collectMetadataFiles(t)

	for _, path := range files {
		filename := filepath.Base(path)
		if filename == baseYAMLFile {
			continue // Already validated by ValidateDependencies
		}

		t.Run(filename, func(t *testing.T) {
			content, err := testMetadataFS.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			var metadata RecipeMetadata
			if err := yaml.Unmarshal(content, &metadata); err != nil {
				t.Fatalf("failed to parse %s: %v", path, err)
			}

			// Build set of components defined in this overlay
			overlayComponents := make(map[string]bool)
			for _, comp := range metadata.Spec.ComponentRefs {
				overlayComponents[comp.Name] = true
			}

			// Check all dependency references
			for _, comp := range metadata.Spec.ComponentRefs {
				for _, dep := range comp.DependencyRefs {
					if !baseComponents[dep] && !overlayComponents[dep] {
						t.Errorf("componentRef %q references unknown dependency %q", comp.Name, dep)
					}
				}
			}
		})
	}
}

// TestAllComponentNamesMatchKnownComponents verifies that all component names
// in recipes match known bundler types.
func TestAllComponentNamesMatchKnownComponents(t *testing.T) {
	files := collectMetadataFiles(t)

	// Get all supported bundle types
	supportedTypes := make(map[string]bool)
	for _, bt := range types.SupportedTypes() {
		supportedTypes[bt.String()] = true
	}

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := testMetadataFS.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			var metadata RecipeMetadata
			if err := yaml.Unmarshal(content, &metadata); err != nil {
				t.Fatalf("failed to parse %s: %v", path, err)
			}

			for _, comp := range metadata.Spec.ComponentRefs {
				if !supportedTypes[comp.Name] {
					t.Errorf("componentRef uses unknown component name %q; valid types: %v",
						comp.Name, getKeys(supportedTypes))
				}
			}
		})
	}
}

// ============================================================================
// Constraint Syntax Tests
// ============================================================================

// TestAllConstraintsSyntaxValid verifies that all constraints use valid syntax:
// - Measurement path format: {type}.{subtype}.{key}
// - Valid operators: >=, <=, >, <, ==, !=, or exact match
func TestAllConstraintsSyntaxValid(t *testing.T) {
	files := collectMetadataFiles(t)

	// Pattern for measurement path: Type.subtype.key (at least 3 parts)
	pathPattern := regexp.MustCompile(`^[A-Za-z0-9]+\.[A-Za-z0-9_/.-]+\.[A-Za-z0-9_/.-]+$`)

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := testMetadataFS.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			var metadata RecipeMetadata
			if err := yaml.Unmarshal(content, &metadata); err != nil {
				t.Fatalf("failed to parse %s: %v", path, err)
			}

			for _, constraint := range metadata.Spec.Constraints {
				// Validate constraint name (measurement path)
				if !pathPattern.MatchString(constraint.Name) {
					t.Errorf("constraint %q has invalid path format; expected {Type}.{subtype}.{key}", constraint.Name)
				}

				// Validate measurement type
				parts := strings.Split(constraint.Name, ".")
				if len(parts) >= 1 {
					measurementType := parts[0]
					if !validMeasurementTypes[measurementType] {
						t.Errorf("constraint %q uses unknown measurement type %q; valid types: %v",
							constraint.Name, measurementType, getKeys(validMeasurementTypes))
					}
				}

				// Validate constraint value (operator + value)
				if err := validateConstraintValue(constraint.Value); err != nil {
					t.Errorf("constraint %q has invalid value %q: %v", constraint.Name, constraint.Value, err)
				}
			}
		})
	}
}

// validateConstraintValue checks if a constraint value has valid syntax.
func validateConstraintValue(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("empty constraint value")
	}

	// Check for operator prefix
	for _, op := range validConstraintOperators {
		if strings.HasPrefix(value, op) {
			remainder := strings.TrimSpace(strings.TrimPrefix(value, op))
			if remainder == "" {
				return fmt.Errorf("operator %q without value", op)
			}
			return nil // Valid operator + value
		}
	}

	// No operator - valid as exact match
	return nil
}

// ============================================================================
// Criteria Uniqueness Tests
// ============================================================================

// TestNoDuplicateCriteriaAcrossOverlays ensures no two overlays have
// identical criteria, which would cause non-deterministic matching.
func TestNoDuplicateCriteriaAcrossOverlays(t *testing.T) {
	files := collectMetadataFiles(t)

	// Map criteria string to file name
	criteriaMap := make(map[string]string)

	for _, path := range files {
		filename := filepath.Base(path)
		if filename == baseYAMLFile {
			continue
		}

		content, err := testMetadataFS.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}

		var metadata RecipeMetadata
		if err := yaml.Unmarshal(content, &metadata); err != nil {
			t.Fatalf("failed to parse %s: %v", path, err)
		}

		// Create criteria key
		c := metadata.Spec.Criteria
		key := fmt.Sprintf("service=%s,accelerator=%s,os=%s,intent=%s",
			c.Service, c.Accelerator, c.OS, c.Intent)

		if existing, found := criteriaMap[key]; found {
			t.Errorf("duplicate criteria found:\n  %s: %s\n  %s: %s",
				existing, key, filename, key)
		}
		criteriaMap[key] = filename
	}
}

// ============================================================================
// Merge Consistency Tests
// ============================================================================

// TestBaseAndOverlaysMergeWithoutConflict verifies that each overlay
// can be merged with base without errors.
func TestBaseAndOverlaysMergeWithoutConflict(t *testing.T) {
	// Load base
	baseContent, err := testMetadataFS.ReadFile("data/" + baseYAMLFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", baseYAMLFile, err)
	}

	var baseMetadata RecipeMetadata
	if err := yaml.Unmarshal(baseContent, &baseMetadata); err != nil {
		t.Fatalf("failed to parse %s: %v", baseYAMLFile, err)
	}

	files := collectMetadataFiles(t)

	for _, path := range files {
		filename := filepath.Base(path)
		if filename == baseYAMLFile {
			continue
		}

		t.Run(filename, func(t *testing.T) {
			content, err := testMetadataFS.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			var overlayMetadata RecipeMetadata
			if err := yaml.Unmarshal(content, &overlayMetadata); err != nil {
				t.Fatalf("failed to parse %s: %v", path, err)
			}

			// Create a copy of base spec for merging
			mergedSpec := baseMetadata.Spec

			// Attempt merge (Merge doesn't return error, panics on nil)
			mergedSpec.Merge(&overlayMetadata.Spec)

			// Verify merge produced valid result
			if len(mergedSpec.ComponentRefs) == 0 {
				t.Error("merged spec has no component refs")
			}
		})
	}
}

// TestMergedRecipesHaveNoCycles verifies that after merging base + overlay,
// the resulting recipe has no circular dependencies.
func TestMergedRecipesHaveNoCycles(t *testing.T) {
	// Load base
	baseContent, err := testMetadataFS.ReadFile("data/" + baseYAMLFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", baseYAMLFile, err)
	}

	var baseMetadata RecipeMetadata
	if err := yaml.Unmarshal(baseContent, &baseMetadata); err != nil {
		t.Fatalf("failed to parse %s: %v", baseYAMLFile, err)
	}

	files := collectMetadataFiles(t)

	for _, path := range files {
		filename := filepath.Base(path)
		if filename == baseYAMLFile {
			continue
		}

		t.Run(filename, func(t *testing.T) {
			content, err := testMetadataFS.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			var overlayMetadata RecipeMetadata
			if err := yaml.Unmarshal(content, &overlayMetadata); err != nil {
				t.Fatalf("failed to parse %s: %v", path, err)
			}

			// Create a copy of base spec for merging
			mergedSpec := baseMetadata.Spec

			// Merge overlay
			mergedSpec.Merge(&overlayMetadata.Spec)

			// Validate no cycles in merged result
			if err := mergedSpec.ValidateDependencies(); err != nil {
				t.Errorf("merged recipe has dependency issues: %v", err)
			}
		})
	}
}

// ============================================================================
// Values File Parsing Tests
// ============================================================================

// TestAllValuesFilesParseAsValidYAML ensures all component values files
// are valid YAML.
func TestAllValuesFilesParseAsValidYAML(t *testing.T) {
	valuesFiles := collectValuesFiles(t)

	for path := range valuesFiles {
		t.Run(path, func(t *testing.T) {
			content, err := testMetadataFS.ReadFile("data/" + path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			// Parse as generic YAML to verify syntax
			var parsed interface{}
			if err := yaml.Unmarshal(content, &parsed); err != nil {
				t.Errorf("failed to parse values file as YAML: %v", err)
			}
		})
	}
}

// ============================================================================
// Base Recipe Validation Tests
// ============================================================================

// TestBaseRecipeValidation verifies the base recipe passes all validations.
func TestBaseRecipeValidation(t *testing.T) {
	content, err := testMetadataFS.ReadFile("data/" + baseYAMLFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", baseYAMLFile, err)
	}

	var metadata RecipeMetadata
	if parseErr := yaml.Unmarshal(content, &metadata); parseErr != nil {
		t.Fatalf("failed to parse %s: %v", baseYAMLFile, parseErr)
	}

	// Validate dependencies
	if depErr := metadata.Spec.ValidateDependencies(); depErr != nil {
		t.Errorf("base recipe dependency validation failed: %v", depErr)
	}

	// Validate topological sort works
	order, sortErr := metadata.Spec.TopologicalSort()
	if sortErr != nil {
		t.Errorf("base recipe topological sort failed: %v", sortErr)
	}

	if len(order) != len(metadata.Spec.ComponentRefs) {
		t.Errorf("topological sort returned %d components, expected %d",
			len(order), len(metadata.Spec.ComponentRefs))
	}
}

// ============================================================================
// Component Type Validation Tests
// ============================================================================

// TestAllComponentTypesValid verifies that all componentRefs use valid types.
func TestAllComponentTypesValid(t *testing.T) {
	files := collectMetadataFiles(t)

	validTypes := map[ComponentType]bool{
		ComponentTypeHelm:      true,
		ComponentTypeKustomize: true,
	}

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := testMetadataFS.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			var metadata RecipeMetadata
			if err := yaml.Unmarshal(content, &metadata); err != nil {
				t.Fatalf("failed to parse %s: %v", path, err)
			}

			for _, comp := range metadata.Spec.ComponentRefs {
				if comp.Type == "" {
					t.Errorf("componentRef %q missing type field", comp.Name)
					continue
				}
				if !validTypes[comp.Type] {
					t.Errorf("componentRef %q has invalid type %q; valid types: Helm, Kustomize",
						comp.Name, comp.Type)
				}
			}
		})
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// collectMetadataFiles returns all YAML files in data/ (excluding components/).
func collectMetadataFiles(t *testing.T) []string {
	t.Helper()

	var files []string
	err := fs.WalkDir(testMetadataFS, "data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Skip component files
		if strings.Contains(path, "components/") {
			return nil
		}
		// Skip non-YAML files
		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		// Skip README
		if strings.HasSuffix(path, "README.md") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk data directory: %v", err)
	}

	return files
}

// collectValuesFiles returns all values files in data/components/.
func collectValuesFiles(t *testing.T) map[string]bool {
	t.Helper()

	files := make(map[string]bool)
	err := fs.WalkDir(testMetadataFS, "data/components", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Store relative path from data/
		relPath := strings.TrimPrefix(path, "data/")
		files[relPath] = true
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk components directory: %v", err)
	}

	return files
}

// getKeys returns the keys of a map as a slice.
func getKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
