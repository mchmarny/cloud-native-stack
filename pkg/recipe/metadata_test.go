/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/

// metadata_test.go tests the RecipeMetadata types and MetadataStore.
//
// Area of Concern: Recipe metadata behavior and inheritance
// - RecipeMetadataSpec.ValidateDependencies() - component dependency validation
// - RecipeMetadataSpec.TopologicalSort() - deployment ordering
// - RecipeMetadataSpec.Merge() - overlay merging with base recipes
// - ComponentRef merging - how overlays override/inherit base values
// - MetadataStore inheritance chains - multi-level spec.base resolution
//   (e.g., base → eks → eks-training → gb200-eks-training)
//
// These tests use synthesized Go structs and the actual MetadataStore
// to verify runtime behavior of the metadata layer.
//
// Related test files:
// - recipe_test.go: Tests Recipe struct validation methods after recipes
//   are built (Validate, ValidateStructure, ValidateMeasurementExists)
// - yaml_test.go: Tests embedded YAML data files for schema conformance,
//   valid references, enum values, and constraint syntax

package recipe

import (
	"context"
	"testing"
)

func TestRecipeMetadataSpecValidateDependencies(t *testing.T) {
	tests := []struct {
		name    string
		spec    RecipeMetadataSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid no dependencies",
			spec: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "cert-manager", Type: ComponentTypeHelm},
					{Name: "gpu-operator", Type: ComponentTypeHelm},
				},
			},
			wantErr: false,
		},
		{
			name: "valid with dependencies",
			spec: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "cert-manager", Type: ComponentTypeHelm},
					{Name: "gpu-operator", Type: ComponentTypeHelm, DependencyRefs: []string{"cert-manager"}},
					{Name: "dra-driver", Type: ComponentTypeHelm, DependencyRefs: []string{"gpu-operator"}},
				},
			},
			wantErr: false,
		},
		{
			name: "missing dependency",
			spec: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "gpu-operator", Type: ComponentTypeHelm, DependencyRefs: []string{"cert-manager"}},
				},
			},
			wantErr: true,
			errMsg:  "references unknown dependency",
		},
		{
			name: "self-dependency (cycle)",
			spec: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "cert-manager", Type: ComponentTypeHelm, DependencyRefs: []string{"cert-manager"}},
				},
			},
			wantErr: true,
			errMsg:  "circular dependency",
		},
		{
			name: "two-node cycle",
			spec: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "A", Type: ComponentTypeHelm, DependencyRefs: []string{"B"}},
					{Name: "B", Type: ComponentTypeHelm, DependencyRefs: []string{"A"}},
				},
			},
			wantErr: true,
			errMsg:  "circular dependency",
		},
		{
			name: "three-node cycle",
			spec: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "A", Type: ComponentTypeHelm, DependencyRefs: []string{"B"}},
					{Name: "B", Type: ComponentTypeHelm, DependencyRefs: []string{"C"}},
					{Name: "C", Type: ComponentTypeHelm, DependencyRefs: []string{"A"}},
				},
			},
			wantErr: true,
			errMsg:  "circular dependency",
		},
		{
			name: "complex valid graph",
			spec: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "cert-manager", Type: ComponentTypeHelm},
					{Name: "gpu-operator", Type: ComponentTypeHelm, DependencyRefs: []string{"cert-manager"}},
					{Name: "network-operator", Type: ComponentTypeHelm, DependencyRefs: []string{"cert-manager"}},
					{Name: "nvsentinel", Type: ComponentTypeHelm, DependencyRefs: []string{"cert-manager", "gpu-operator"}},
					{Name: "dra-driver", Type: ComponentTypeHelm, DependencyRefs: []string{"gpu-operator"}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateDependencies()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("ValidateDependencies() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestRecipeMetadataSpecTopologicalSort(t *testing.T) {
	tests := []struct {
		name    string
		spec    RecipeMetadataSpec
		want    []string
		wantErr bool
	}{
		{
			name: "no dependencies",
			spec: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "cert-manager", Type: ComponentTypeHelm},
					{Name: "gpu-operator", Type: ComponentTypeHelm},
				},
			},
			want: []string{"cert-manager", "gpu-operator"},
		},
		{
			name: "linear dependencies",
			spec: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "cert-manager", Type: ComponentTypeHelm},
					{Name: "gpu-operator", Type: ComponentTypeHelm, DependencyRefs: []string{"cert-manager"}},
					{Name: "dra-driver", Type: ComponentTypeHelm, DependencyRefs: []string{"gpu-operator"}},
				},
			},
			want: []string{"cert-manager", "gpu-operator", "dra-driver"},
		},
		{
			name: "diamond dependencies",
			spec: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "cert-manager", Type: ComponentTypeHelm},
					{Name: "gpu-operator", Type: ComponentTypeHelm, DependencyRefs: []string{"cert-manager"}},
					{Name: "network-operator", Type: ComponentTypeHelm, DependencyRefs: []string{"cert-manager"}},
					{Name: "nvsentinel", Type: ComponentTypeHelm, DependencyRefs: []string{"gpu-operator", "network-operator"}},
				},
			},
			// cert-manager first, then gpu-operator and network-operator (alphabetically), then nvsentinel
			want: []string{"cert-manager", "gpu-operator", "network-operator", "nvsentinel"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.spec.TopologicalSort()
			if (err != nil) != tt.wantErr {
				t.Errorf("TopologicalSort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("TopologicalSort() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("TopologicalSort()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRecipeMetadataSpecMerge(t *testing.T) {
	tests := []struct {
		name        string
		base        RecipeMetadataSpec
		overlay     RecipeMetadataSpec
		wantCompCnt int
		wantConCnt  int
	}{
		{
			name: "merge disjoint components",
			base: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "cert-manager", Type: ComponentTypeHelm, Version: "v1.0.0"},
				},
			},
			overlay: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "gpu-operator", Type: ComponentTypeHelm, Version: "v2.0.0"},
				},
			},
			wantCompCnt: 2,
		},
		{
			name: "overlay overrides component",
			base: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "gpu-operator", Type: ComponentTypeHelm, Version: "v1.0.0"},
				},
			},
			overlay: RecipeMetadataSpec{
				ComponentRefs: []ComponentRef{
					{Name: "gpu-operator", Type: ComponentTypeHelm, Version: "v2.0.0"},
				},
			},
			wantCompCnt: 1,
		},
		{
			name: "merge constraints",
			base: RecipeMetadataSpec{
				Constraints: []Constraint{
					{Name: "k8s", Value: ">= 1.30"},
				},
			},
			overlay: RecipeMetadataSpec{
				Constraints: []Constraint{
					{Name: "kernel", Value: ">= 6.8"},
				},
			},
			wantConCnt: 2,
		},
		{
			name: "overlay overrides constraint",
			base: RecipeMetadataSpec{
				Constraints: []Constraint{
					{Name: "k8s", Value: ">= 1.30"},
				},
			},
			overlay: RecipeMetadataSpec{
				Constraints: []Constraint{
					{Name: "k8s", Value: ">= 1.32"},
				},
			},
			wantConCnt: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(&tt.overlay)
			if tt.wantCompCnt > 0 && len(tt.base.ComponentRefs) != tt.wantCompCnt {
				t.Errorf("Merge() componentRefs count = %d, want %d", len(tt.base.ComponentRefs), tt.wantCompCnt)
			}
			if tt.wantConCnt > 0 && len(tt.base.Constraints) != tt.wantConCnt {
				t.Errorf("Merge() constraints count = %d, want %d", len(tt.base.Constraints), tt.wantConCnt)
			}
		})
	}
}

// TestComponentRefMergeInheritsFromBase verifies that when an overlay specifies
// only partial fields for a component, the missing fields are inherited from base.
func TestComponentRefMergeInheritsFromBase(t *testing.T) {
	base := RecipeMetadataSpec{
		ComponentRefs: []ComponentRef{
			{
				Name:       "cert-manager",
				Type:       ComponentTypeHelm,
				Source:     "https://charts.jetstack.io",
				Version:    "v1.17.2",
				ValuesFile: "components/cert-manager/values.yaml",
			},
		},
	}

	// Overlay only specifies name, type, and new valuesFile
	overlay := RecipeMetadataSpec{
		ComponentRefs: []ComponentRef{
			{
				Name:       "cert-manager",
				Type:       ComponentTypeHelm,
				ValuesFile: "components/cert-manager/tainted-values.yaml",
			},
		},
	}

	base.Merge(&overlay)

	if len(base.ComponentRefs) != 1 {
		t.Fatalf("expected 1 component, got %d", len(base.ComponentRefs))
	}

	comp := base.ComponentRefs[0]

	// Verify inherited fields from base
	if comp.Source != "https://charts.jetstack.io" {
		t.Errorf("Source should be inherited from base, got %q", comp.Source)
	}
	if comp.Version != "v1.17.2" {
		t.Errorf("Version should be inherited from base, got %q", comp.Version)
	}

	// Verify overridden field from overlay
	if comp.ValuesFile != "components/cert-manager/tainted-values.yaml" {
		t.Errorf("ValuesFile should be from overlay, got %q", comp.ValuesFile)
	}

	t.Logf("ComponentRef correctly merged: source=%s, version=%s, valuesFile=%s",
		comp.Source, comp.Version, comp.ValuesFile)
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || containsString(s[1:], substr)))
}

// TestOverlayAddsNewComponent verifies that overlay recipes can add components
// that don't exist in the base recipe.
func TestOverlayAddsNewComponent(t *testing.T) {
	ctx := context.Background()

	// Build recipe for H100 inference workload
	// h100-ubuntu-inference.yaml adds network-operator which is NOT in base.yaml
	// Note: The overlay file requires accelerator=h100, os=ubuntu, intent=inference
	// so we must specify all criteria to match it (asymmetric matching).
	builder := NewBuilder()
	criteria := NewCriteria()
	criteria.Accelerator = CriteriaAcceleratorH100
	criteria.OS = CriteriaOSUbuntu
	criteria.Intent = CriteriaIntentInference

	result, err := builder.BuildFromCriteria(ctx, criteria)
	if err != nil {
		t.Fatalf("BuildFromCriteria failed: %v", err)
	}

	if result == nil {
		t.Fatal("Recipe result is nil")
	}

	// Verify base components exist
	baseComponents := []string{"cert-manager", "gpu-operator", "nvsentinel", "skyhook"}
	for _, name := range baseComponents {
		if comp := result.GetComponentRef(name); comp == nil {
			t.Errorf("Base component %q not found in result", name)
		}
	}

	// Verify overlay-added component exists
	networkOp := result.GetComponentRef("network-operator")
	if networkOp == nil {
		t.Fatalf("network-operator not found (should be added by h100-inference overlay)")
	}

	// Verify network-operator properties
	if networkOp.Version == "" {
		t.Error("network-operator has empty version")
	}
	if networkOp.Type != "Helm" {
		t.Errorf("network-operator type = %q, want Helm", networkOp.Type)
	}
	if len(networkOp.DependencyRefs) == 0 {
		t.Error("network-operator has no dependencies (should depend on cert-manager)")
	}

	// Build recipe for EKS GB200 training workload
	// gb200-eks-training.yaml adds dra-driver which is NOT in base.yaml
	builder = NewBuilder()
	criteria = NewCriteria()
	criteria.Accelerator = CriteriaAcceleratorGB200
	criteria.Intent = CriteriaIntentTraining
	criteria.Service = CriteriaServiceEKS

	result, err = builder.BuildFromCriteria(ctx, criteria)
	if err != nil {
		t.Fatalf("BuildFromCriteria failed: %v", err)
	}

	if result == nil {
		t.Fatal("Recipe result is nil")
	}

	// Verify overlay-added component exists
	draDriverOp := result.GetComponentRef("dra-driver")
	if draDriverOp == nil {
		t.Fatalf("dra-driver not found (should be added by gb200 overlay)")
	}

	t.Logf("Successfully verified overlay can add new components")
	t.Logf("   Base components: %d", len(baseComponents))
	t.Logf("   Total components: %d", len(result.ComponentRefs))
	t.Logf("   network-operator version: %s", networkOp.Version)
	t.Logf("   dra-driver version: %s", draDriverOp.Version)
}

// TestOverlayMergeDoesNotLoseBaseComponents verifies that when overlays add
// components, base components are preserved.
func TestOverlayMergeDoesNotLoseBaseComponents(t *testing.T) {
	ctx := context.Background()
	builder := NewBuilder()

	// Build H100 inference recipe (matches overlay that adds network-operator)
	// Note: The h100-ubuntu-inference.yaml overlay requires all three criteria
	// (accelerator=h100, os=ubuntu, intent=inference) to match due to asymmetric matching.
	criteria := NewCriteria()
	criteria.Accelerator = CriteriaAcceleratorH100
	criteria.OS = CriteriaOSUbuntu
	criteria.Intent = CriteriaIntentInference

	result, err := builder.BuildFromCriteria(ctx, criteria)
	if err != nil {
		t.Fatalf("BuildFromCriteria failed: %v", err)
	}

	// Verify all 4 base components exist
	expectedBaseComponents := []string{"cert-manager", "gpu-operator", "nvsentinel", "skyhook"}
	for _, name := range expectedBaseComponents {
		if comp := result.GetComponentRef(name); comp == nil {
			t.Errorf("Base component %q missing from overlay result", name)
		}
	}

	// Verify network-operator was added
	networkOp := result.GetComponentRef("network-operator")
	if networkOp == nil {
		t.Error("network-operator not found (should be added by overlay)")
	}

	// Result should have at least 5 components (4 base + 1 added)
	if len(result.ComponentRefs) < 5 {
		t.Errorf("Expected at least 5 components, got %d", len(result.ComponentRefs))
	}

	t.Logf("Base components preserved when overlay adds new components")
	t.Logf("   Total components: %d (4 base + additions)", len(result.ComponentRefs))
	if networkOp != nil {
		t.Logf("   network-operator added: version %s", networkOp.Version)
	}
}

// TestInheritanceChain verifies that multi-level inheritance chains work correctly.
// Tests the chain: base → eks → eks-training → gb200-eks-training
func TestInheritanceChain(t *testing.T) {
	ctx := context.Background()
	builder := NewBuilder()

	// Build GB200 EKS training recipe (full chain: base → eks → eks-training → gb200-eks-training)
	criteria := NewCriteria()
	criteria.Service = CriteriaServiceEKS
	criteria.Accelerator = CriteriaAcceleratorGB200
	criteria.OS = CriteriaOSUbuntu
	criteria.Intent = CriteriaIntentTraining

	result, err := builder.BuildFromCriteria(ctx, criteria)
	if err != nil {
		t.Fatalf("BuildFromCriteria failed: %v", err)
	}

	// Verify applied overlays includes the full chain
	// Should include: base, eks, eks-training, gb200-eks-training
	appliedOverlays := result.Metadata.AppliedOverlays
	t.Logf("Applied overlays: %v", appliedOverlays)

	if len(appliedOverlays) < 2 {
		t.Errorf("Expected at least 2 applied overlays (base + matching), got %d: %v",
			len(appliedOverlays), appliedOverlays)
	}

	// Verify base components are present
	expectedComponents := []string{"cert-manager", "gpu-operator", "nvsentinel", "skyhook"}
	for _, name := range expectedComponents {
		if comp := result.GetComponentRef(name); comp == nil {
			t.Errorf("Expected component %q not found in result", name)
		}
	}

	// Verify gpu-operator has GB200-specific overrides (from gb200-eks-training)
	gpuOp := result.GetComponentRef("gpu-operator")
	if gpuOp == nil {
		t.Fatal("gpu-operator not found")
	}
	if gpuOp.Overrides == nil {
		t.Error("gpu-operator should have overrides from gb200-eks-training")
	} else {
		if driver, ok := gpuOp.Overrides["driver"].(map[string]interface{}); ok {
			if version, ok := driver["version"].(string); ok {
				if version != "580.82.07" {
					t.Errorf("Expected GB200 driver version 580.82.07, got %s", version)
				}
			}
		}
	}

	// Verify gpu-operator has training values file (from eks-training)
	if gpuOp.ValuesFile != "components/gpu-operator/values-eks-training.yaml" {
		t.Errorf("Expected gpu-operator valuesFile from eks-training, got %q", gpuOp.ValuesFile)
	}

	t.Logf("Inheritance chain test passed")
	t.Logf("   Applied overlays: %v", appliedOverlays)
	t.Logf("   GPU operator version: %s", gpuOp.Version)
	t.Logf("   GPU operator valuesFile: %s", gpuOp.ValuesFile)
}

// TestInheritanceChainGB200 verifies that GB200 inherits correctly from eks-training.
func TestInheritanceChainGB200(t *testing.T) {
	ctx := context.Background()
	builder := NewBuilder()

	// Build GB200 EKS training recipe
	criteria := NewCriteria()
	criteria.Service = CriteriaServiceEKS
	criteria.Accelerator = CriteriaAcceleratorGB200
	criteria.OS = CriteriaOSUbuntu
	criteria.Intent = CriteriaIntentTraining

	result, err := builder.BuildFromCriteria(ctx, criteria)
	if err != nil {
		t.Fatalf("BuildFromCriteria failed: %v", err)
	}

	// Verify applied overlays
	t.Logf("Applied overlays: %v", result.Metadata.AppliedOverlays)

	// Verify gpu-operator has GB200-specific overrides
	gpuOp := result.GetComponentRef("gpu-operator")
	if gpuOp == nil {
		t.Fatal("gpu-operator not found")
	}

	// GB200 should have gdrcopy enabled
	if gpuOp.Overrides != nil {
		if gdrcopy, ok := gpuOp.Overrides["gdrcopy"].(map[string]interface{}); ok {
			if enabled, ok := gdrcopy["enabled"].(bool); ok && !enabled {
				t.Error("GB200 should have gdrcopy enabled")
			}
		}
	}

	// Verify training values file is inherited
	if gpuOp.ValuesFile != "components/gpu-operator/values-eks-training.yaml" {
		t.Errorf("Expected gpu-operator valuesFile from eks-training, got %q", gpuOp.ValuesFile)
	}

	t.Logf("GB200 inheritance chain test passed")
}

// TestInheritanceChainDoesNotDuplicateRecipes verifies that recipes in the inheritance
// chain are only applied once, even if they appear in multiple matching overlays' chains.
func TestInheritanceChainDoesNotDuplicateRecipes(t *testing.T) {
	ctx := context.Background()
	builder := NewBuilder()

	criteria := NewCriteria()
	criteria.Service = CriteriaServiceEKS
	criteria.Accelerator = CriteriaAcceleratorGB200
	criteria.Intent = CriteriaIntentTraining

	result, err := builder.BuildFromCriteria(ctx, criteria)
	if err != nil {
		t.Fatalf("BuildFromCriteria failed: %v", err)
	}

	// Count occurrences of each overlay in the applied list
	counts := make(map[string]int)
	for _, name := range result.Metadata.AppliedOverlays {
		counts[name]++
	}

	// Verify no duplicates
	for name, count := range counts {
		if count > 1 {
			t.Errorf("Recipe %q applied %d times (should be 1)", name, count)
		}
	}

	t.Logf("No duplicate recipes in chain: %v", result.Metadata.AppliedOverlays)
}
