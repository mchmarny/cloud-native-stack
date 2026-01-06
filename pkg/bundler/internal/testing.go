package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe/header"
)

// BundlerInterface defines the interface that bundlers must implement for testing.
type BundlerInterface interface {
	Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*result.Result, error)
}

// TestHarness provides common testing utilities for bundlers.
type TestHarness struct {
	t             *testing.T
	bundlerName   string
	expectedFiles []string
	recipeBuilder func() *RecipeBuilder
}

// NewTestHarness creates a new test harness for a bundler.
func NewTestHarness(t *testing.T, bundlerName string) *TestHarness {
	return &TestHarness{
		t:             t,
		bundlerName:   bundlerName,
		expectedFiles: []string{},
	}
}

// WithExpectedFiles sets the list of files expected to be generated.
func (h *TestHarness) WithExpectedFiles(files []string) *TestHarness {
	h.expectedFiles = files
	return h
}

// WithRecipeBuilder sets a custom recipe builder function.
func (h *TestHarness) WithRecipeBuilder(builder func() *RecipeBuilder) *TestHarness {
	h.recipeBuilder = builder
	return h
}

// TestMake tests the Make method of a bundler with standard assertions.
func (h *TestHarness) TestMake(bundler BundlerInterface) {
	ctx := context.Background()
	tmpDir := h.t.TempDir()

	rec := h.getRecipe()
	result, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		h.t.Fatalf("Make() error = %v", err)
	}

	h.AssertResult(result, tmpDir)
}

// TestMakeWithConfig tests the Make method with a specific config.
func (h *TestHarness) TestMakeWithConfig(bundler BundlerInterface, _ *config.Config) {
	ctx := context.Background()
	tmpDir := h.t.TempDir()

	rec := h.getRecipe()
	result, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		h.t.Fatalf("Make() error = %v", err)
	}

	h.AssertResult(result, tmpDir)
}

// AssertResult performs standard assertions on a bundler result.
func (h *TestHarness) AssertResult(result *result.Result, outputDir string) {
	if result == nil {
		h.t.Fatal("Make() returned nil result")
		return
	}

	if !result.Success {
		h.t.Error("Make() should succeed")
	}

	if len(result.Files) == 0 {
		h.t.Error("Make() produced no files")
	}

	// Verify bundle directory structure
	bundleDir := filepath.Join(outputDir, h.bundlerName)
	if _, err := os.Stat(bundleDir); os.IsNotExist(err) {
		h.t.Errorf("Make() did not create %s directory", h.bundlerName)
	}

	// Verify expected files exist
	for _, file := range h.expectedFiles {
		path := filepath.Join(bundleDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			h.t.Errorf("Expected file %s not found", file)
		}
	}
}

// AssertFileExists checks if a file exists in the bundle directory.
func (h *TestHarness) AssertFileExists(outputDir, filename string) {
	bundleDir := filepath.Join(outputDir, h.bundlerName)
	path := filepath.Join(bundleDir, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		h.t.Errorf("Expected file %s not found", filename)
	}
}

// getRecipe returns a test recipe, using custom builder if set.
func (h *TestHarness) getRecipe() *recipe.Recipe {
	if h.recipeBuilder != nil {
		return h.recipeBuilder().Build()
	}
	return h.createDefaultRecipe()
}

// createDefaultRecipe creates a basic recipe for testing.
func (h *TestHarness) createDefaultRecipe() *recipe.Recipe {
	r := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "config",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}
	r.Init(header.KindRecipe, "v1")
	return r
}

// TestTemplateGetter tests a template getter function.
func TestTemplateGetter(t *testing.T, getTemplate func(string) (string, bool), expectedTemplates []string) {
	for _, name := range expectedTemplates {
		t.Run(name, func(t *testing.T) {
			tmpl, ok := getTemplate(name)
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
		_, ok := getTemplate("nonexistent")
		if ok {
			t.Error("GetTemplate() should return false for non-existent template")
		}
	})
}

// RecipeBuilder helps build test recipes with fluent API.
type RecipeBuilder struct {
	measurements []*measurement.Measurement
}

// NewRecipeBuilder creates a new recipe builder.
func NewRecipeBuilder() *RecipeBuilder {
	return &RecipeBuilder{
		measurements: []*measurement.Measurement{},
	}
}

// WithK8sMeasurement adds a K8s measurement with the given subtypes.
func (rb *RecipeBuilder) WithK8sMeasurement(subtypes ...measurement.Subtype) *RecipeBuilder {
	rb.measurements = append(rb.measurements, &measurement.Measurement{
		Type:     measurement.TypeK8s,
		Subtypes: subtypes,
	})
	return rb
}

// WithGPUMeasurement adds a GPU measurement with the given subtypes.
func (rb *RecipeBuilder) WithGPUMeasurement(subtypes ...measurement.Subtype) *RecipeBuilder {
	rb.measurements = append(rb.measurements, &measurement.Measurement{
		Type:     measurement.TypeGPU,
		Subtypes: subtypes,
	})
	return rb
}

// WithOSMeasurement adds an OS measurement with the given subtypes.
func (rb *RecipeBuilder) WithOSMeasurement(subtypes ...measurement.Subtype) *RecipeBuilder {
	rb.measurements = append(rb.measurements, &measurement.Measurement{
		Type:     measurement.TypeOS,
		Subtypes: subtypes,
	})
	return rb
}

// WithSystemDMeasurement adds a SystemD measurement with the given subtypes.
func (rb *RecipeBuilder) WithSystemDMeasurement(subtypes ...measurement.Subtype) *RecipeBuilder {
	rb.measurements = append(rb.measurements, &measurement.Measurement{
		Type:     measurement.TypeSystemD,
		Subtypes: subtypes,
	})
	return rb
}

// Build creates the recipe.
func (rb *RecipeBuilder) Build() *recipe.Recipe {
	r := &recipe.Recipe{
		Measurements: rb.measurements,
	}
	r.Init(header.KindRecipe, "v1")
	return r
}

// ImageSubtype creates an image subtype with common image data.
func ImageSubtype(images map[string]string) measurement.Subtype {
	data := make(map[string]measurement.Reading)
	for k, v := range images {
		data[k] = measurement.Str(v)
	}
	return measurement.Subtype{
		Name: "image",
		Data: data,
	}
}

// RegistrySubtype creates a registry subtype with registry information.
func RegistrySubtype(registry map[string]string) measurement.Subtype {
	data := make(map[string]measurement.Reading)
	for k, v := range registry {
		data[k] = measurement.Str(v)
	}
	return measurement.Subtype{
		Name: "registry",
		Data: data,
	}
}

// ConfigSubtype creates a config subtype with common config data.
func ConfigSubtype(configs map[string]interface{}) measurement.Subtype {
	data := make(map[string]measurement.Reading)
	for k, v := range configs {
		switch val := v.(type) {
		case string:
			data[k] = measurement.Str(val)
		case bool:
			data[k] = measurement.Bool(val)
		case int:
			data[k] = measurement.Int(val)
		case float64:
			data[k] = measurement.Float64(val)
		}
	}
	return measurement.Subtype{
		Name: "config",
		Data: data,
	}
}

// SMISubtype creates an SMI subtype for GPU measurements.
func SMISubtype(data map[string]string) measurement.Subtype {
	readings := make(map[string]measurement.Reading)
	for k, v := range data {
		readings[k] = measurement.Str(v)
	}
	return measurement.Subtype{
		Name: "smi",
		Data: readings,
	}
}

// GrubSubtype creates a grub subtype for OS measurements.
func GrubSubtype(data map[string]string) measurement.Subtype {
	readings := make(map[string]measurement.Reading)
	for k, v := range data {
		readings[k] = measurement.Str(v)
	}
	return measurement.Subtype{
		Name: "grub",
		Data: readings,
	}
}

// SysctlSubtype creates a sysctl subtype for OS measurements.
func SysctlSubtype(data map[string]string) measurement.Subtype {
	readings := make(map[string]measurement.Reading)
	for k, v := range data {
		readings[k] = measurement.Str(v)
	}
	return measurement.Subtype{
		Name: "sysctl",
		Data: readings,
	}
}

// ServiceSubtype creates a service subtype for SystemD measurements.
func ServiceSubtype(serviceName string, data map[string]string) measurement.Subtype {
	readings := make(map[string]measurement.Reading)
	for k, v := range data {
		readings[k] = measurement.Str(v)
	}
	return measurement.Subtype{
		Name: serviceName,
		Data: readings,
	}
}

// TestValidateRecipe is a reusable test for recipe validation.
func TestValidateRecipe(t *testing.T, validateFunc func(*recipe.Recipe) error) {
	tests := []struct {
		name    string
		recipe  *recipe.Recipe
		wantErr bool
	}{
		{
			name:    "nil recipe",
			recipe:  nil,
			wantErr: true,
		},
		{
			name: "empty measurements",
			recipe: &recipe.Recipe{
				Measurements: []*measurement.Measurement{},
			},
			wantErr: true,
		},
		{
			name: "valid recipe",
			recipe: NewRecipeBuilder().
				WithK8sMeasurement(ConfigSubtype(map[string]interface{}{
					"version": "1.28.0",
				})).
				Build(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFunc(tt.recipe)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRecipe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// AssertConfigValue checks if a config value matches the expected value.
func AssertConfigValue(t *testing.T, config map[string]string, key, expected string) {
	t.Helper()
	if val, ok := config[key]; !ok {
		t.Errorf("Config missing key %s", key)
	} else if val != expected {
		t.Errorf("Config[%s] = %s, want %s", key, val, expected)
	}
}

// AssertValueWithContext checks if a ValueWithContext has the expected value.
func AssertValueWithContext(t *testing.T, vwc ValueWithContext, expected interface{}, fieldName string) {
	t.Helper()
	if vwc.Value != expected {
		t.Errorf("%s = %v, want %v", fieldName, vwc.Value, expected)
	}
}
