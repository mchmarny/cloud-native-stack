package certmanager

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

func TestNewBundler(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "with nil config",
			cfg:  nil,
		},
		{
			name: "with valid config",
			cfg:  config.NewConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBundler(tt.cfg)
			if b == nil {
				t.Fatal("NewBundler() returned nil")
			}
			if b.Config == nil {
				t.Error("Bundler config should not be nil")
			}
		})
	}
}

func TestBundler_Make(t *testing.T) {
	tests := []struct {
		name       string
		recipe     *recipe.RecipeResult
		wantErr    bool
		verifyFunc func(t *testing.T, outputDir string)
	}{
		{
			name:    "valid recipe with cert-manager component",
			recipe:  createTestRecipeResult(),
			wantErr: false,
			verifyFunc: func(t *testing.T, outputDir string) {
				bundleDir := filepath.Join(outputDir, "cert-manager")

				// Verify values.yaml exists
				valuesPath := filepath.Join(bundleDir, "values.yaml")
				if _, err := os.Stat(valuesPath); os.IsNotExist(err) {
					t.Errorf("Expected values.yaml not found")
				}

				// Verify README
				readmePath := filepath.Join(bundleDir, "README.md")
				if _, err := os.Stat(readmePath); os.IsNotExist(err) {
					t.Errorf("Expected README.md not found")
				}

				// Verify checksums.txt
				checksumPath := filepath.Join(bundleDir, "checksums.txt")
				if _, err := os.Stat(checksumPath); os.IsNotExist(err) {
					t.Errorf("Expected checksums.txt not found")
				}
			},
		},
		{
			name: "recipe with inline overrides",
			recipe: createTestRecipeResultWithOverrides(map[string]interface{}{
				"driver": map[string]interface{}{
					"version": "585.0.0",
				},
			}),
			wantErr: false,
			verifyFunc: func(t *testing.T, outputDir string) {
				bundleDir := filepath.Join(outputDir, "cert-manager")
				valuesPath := filepath.Join(bundleDir, "values.yaml")

				data, err := os.ReadFile(valuesPath)
				if err != nil {
					t.Fatalf("Failed to read values.yaml: %v", err)
				}

				content := string(data)
				if !strings.Contains(content, "585.0.0") {
					t.Error("Expected overridden driver version 585.0.0 not found")
				}
			},
		},
		{
			name:    "missing cert-manager component",
			recipe:  createRecipeResultWithoutGPUOperator(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			b := NewBundler(nil)
			ctx := context.Background()

			result, err := b.Make(ctx, tt.recipe, tmpDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("Make() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("Make() returned nil result")
					return
				}
				if len(result.Files) == 0 {
					t.Error("Make() returned no files")
				}

				if tt.verifyFunc != nil {
					tt.verifyFunc(t, tmpDir)
				}
			}
		})
	}
}

func TestGetTemplate(t *testing.T) {
	expectedTemplates := []string{
		"README.md",
	}

	for _, name := range expectedTemplates {
		t.Run(name, func(t *testing.T) {
			tmpl, ok := GetTemplate(name)
			if !ok {
				t.Errorf("GetTemplate(%s) not found", name)
			}
			if tmpl == "" {
				t.Errorf("GetTemplate(%s) returned empty template", name)
			}
		})
	}

	// Test non-existent template
	t.Run("nonexistent", func(t *testing.T) {
		_, ok := GetTemplate("nonexistent")
		if ok {
			t.Error("GetTemplate() should return false for non-existent template")
		}
	})
}

// Helper function to create a test RecipeResult
func createTestRecipeResult() *recipe.RecipeResult {
	return &recipe.RecipeResult{
		Kind:       "recipeResult",
		APIVersion: recipe.FullAPIVersion,
		ComponentRefs: []recipe.ComponentRef{
			{
				Name:    "cert-manager",
				Type:    "Helm",
				Source:  "https://helm.ngc.nvidia.com/nvidia",
				Version: "v25.3.4",
				// Use inline overrides instead of ValuesFile for testing
				Overrides: map[string]interface{}{
					"operator": map[string]interface{}{
						"version": "v25.3.4",
					},
					"driver": map[string]interface{}{
						"version": "580.0.0",
					},
					"mig": map[string]interface{}{
						"strategy": "mixed",
					},
					"gds": map[string]interface{}{
						"enabled": false,
					},
				},
			},
		},
	}
}

// Helper function to create a test RecipeResult with overrides
func createTestRecipeResultWithOverrides(overrides map[string]interface{}) *recipe.RecipeResult {
	// Start with base values including required fields
	baseValues := map[string]interface{}{
		"operator": map[string]interface{}{
			"version": "v25.3.4",
		},
		"driver": map[string]interface{}{
			"version": "580.0.0",
		},
		"mig": map[string]interface{}{
			"strategy": "mixed",
		},
		"gds": map[string]interface{}{
			"enabled": false,
		},
	}

	// Merge overrides into base
	for k, v := range overrides {
		baseValues[k] = v
	}

	return &recipe.RecipeResult{
		Kind:       "recipeResult",
		APIVersion: recipe.FullAPIVersion,
		ComponentRefs: []recipe.ComponentRef{
			{
				Name:    "cert-manager",
				Type:    "Helm",
				Source:  "https://helm.ngc.nvidia.com/nvidia",
				Version: "v25.3.4",
				// Use inline overrides for testing
				Overrides: baseValues,
			},
		},
	}
}

// Helper function to create a RecipeResult without cert-manager
func createRecipeResultWithoutGPUOperator() *recipe.RecipeResult {
	return &recipe.RecipeResult{
		Kind:       "recipeResult",
		APIVersion: recipe.FullAPIVersion,
		ComponentRefs: []recipe.ComponentRef{
			{
				Name:    "other-component",
				Type:    "Helm",
				Version: "v1.0.0",
			},
		},
	}
}
