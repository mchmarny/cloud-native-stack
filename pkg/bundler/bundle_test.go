package bundler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// mockBundler for testing.
type mockBundler struct {
	shouldFail bool
}

func (m *mockBundler) Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*BundleResult, error) {
	result := NewBundleResult("mock")
	if m.shouldFail {
		return result, errors.New(errors.ErrCodeInternal, "mock bundler failed")
	}
	result.AddFile("test.txt", 100)
	result.MarkSuccess()
	return result, nil
}

// mockValidatingBundler for testing validation.
type mockValidatingBundler struct {
	shouldFail bool
}

func (m *mockValidatingBundler) Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*BundleResult, error) {
	result := NewBundleResult("mock-validator")
	result.AddFile("test.txt", 100)
	result.MarkSuccess()
	return result, nil
}

func (m *mockValidatingBundler) Validate(_ context.Context, _ *recipe.Recipe) error {
	if m.shouldFail {
		return errors.New(errors.ErrCodeInvalidRequest, "validation failed")
	}
	return nil
}

// mockConfigurableBundler for testing configuration.
type mockConfigurableBundler struct {
	config *BundlerConfig
}

func (m *mockConfigurableBundler) Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*BundleResult, error) {
	result := NewBundleResult("mock-configurable")
	result.AddFile("test.txt", 100)
	result.MarkSuccess()
	return result, nil
}

func (m *mockConfigurableBundler) Configure(config *BundlerConfig) error {
	if config == nil {
		return errors.New(errors.ErrCodeInvalidRequest, "config cannot be nil")
	}
	m.config = config
	return nil
}

func init() {
	defaultRegistry.Register("mock", &mockBundler{})
	defaultRegistry.Register("mock-validator", &mockValidatingBundler{})
	defaultRegistry.Register("mock-configurable", &mockConfigurableBundler{})
	defaultRegistry.Register("mock-fail", &mockBundler{shouldFail: true})
}

func TestDefaultBundler_Make(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create a test recipe
	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	bundler := New(WithBundlerTypes([]BundleType{"mock"}))
	output, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if output == nil {
		t.Fatal("Make() returned nil output")
	}

	if len(output.Results) == 0 {
		t.Error("Make() produced no results")
	}

	if output.OutputDir != tmpDir {
		t.Errorf("OutputDir = %s, want %s", output.OutputDir, tmpDir)
	}

	if output.TotalFiles != 1 {
		t.Errorf("TotalFiles = %d, want 1", output.TotalFiles)
	}

	if output.TotalSize != 100 {
		t.Errorf("TotalSize = %d, want 100", output.TotalSize)
	}
}

func TestDefaultBundler_MakeWithNilRecipe(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	bundler := New()
	_, err := bundler.Make(ctx, nil, tmpDir)
	if err == nil {
		t.Error("Make() with nil recipe should return error")
	}
}

func TestDefaultBundler_MakeWithEmptyMeasurements(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{},
	}

	bundler := New()
	_, err := bundler.Make(ctx, rec, tmpDir)
	if err == nil {
		t.Error("Make() with empty measurements should return error")
	}
}

func TestDefaultBundler_MakeWithOptions(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	config := DefaultBundlerConfig()
	config.Namespace = "test-namespace"

	bundler := New(
		WithBundlerTypes([]BundleType{"mock"}),
		WithConfig(config),
	)
	output, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if output == nil {
		t.Fatal("Make() returned nil output")
	}
}

func TestDefaultBundler_MakeCreatesDirectory(t *testing.T) {
	ctx := context.Background()
	tmpDir := filepath.Join(t.TempDir(), "nested", "dir")

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	bundler := New(WithBundlerTypes([]BundleType{"mock"}))
	_, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Make() did not create output directory")
	}
}

func TestDefaultBundler_MakeWithDryRun(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	bundler := New(
		WithBundlerTypes([]BundleType{"mock"}),
		WithDryRun(true),
	)
	output, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if output == nil {
		t.Fatal("Make() returned nil output")
	}

	// In dry run mode, no files should be created
	if output.TotalFiles > 0 {
		t.Errorf("DryRun should not create files, got %d files", output.TotalFiles)
	}
}

func TestDefaultBundler_MakeWithParallel(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	// Parallel is the default (Sequential=false)
	bundler := New(
		WithBundlerTypes([]BundleType{"mock", "mock-validator"}),
	)
	output, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if output == nil {
		t.Fatal("Make() returned nil output")
	}

	// Should have results from both bundlers
	if len(output.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(output.Results))
	}
}

func TestDefaultBundler_MakeWithSequential(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	// Explicitly enable sequential execution
	bundler := New(
		WithBundlerTypes([]BundleType{"mock", "mock-validator"}),
		WithSequential(true),
	)
	output, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if output == nil {
		t.Fatal("Make() returned nil output")
	}

	// Should have results from both bundlers executed sequentially
	if len(output.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(output.Results))
	}
}

func TestDefaultBundler_MakeWithFailFast(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	bundler := New(
		WithBundlerTypes([]BundleType{"mock-fail", "mock"}),
		WithFailFast(true),
	)
	_, err := bundler.Make(ctx, rec, tmpDir)
	if err == nil {
		t.Error("Expected error with FailFast and failing bundler")
	}
}

func TestDefaultBundler_MakeWithoutFailFast(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	bundler := New(
		WithBundlerTypes([]BundleType{"mock-fail", "mock"}),
		WithFailFast(false),
	)
	output, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	// Should collect errors but continue execution
	if !output.HasErrors() {
		t.Error("Expected errors to be collected")
	}

	// Should still have results from successful bundler
	if len(output.Results) == 0 {
		t.Error("Expected at least one result")
	}
}

func TestDefaultBundler_MakeWithValidation(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	bundler := New(WithBundlerTypes([]BundleType{"mock-validator"}))
	output, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if output == nil {
		t.Fatal("Make() returned nil output")
	}
}

func TestDefaultBundler_MakeWithValidationFailure(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	// Register a failing validator
	defaultRegistry.Register("mock-validator-fail", &mockValidatingBundler{shouldFail: true})

	bundler := New(
		WithBundlerTypes([]BundleType{"mock-validator-fail"}),
		WithFailFast(true), // Enable fail-fast to get error
	)
	_, err := bundler.Make(ctx, rec, tmpDir)
	if err == nil {
		t.Error("Expected validation error")
	}
}

func TestDefaultBundler_MakeWithConfiguration(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	config := DefaultBundlerConfig()
	config.Namespace = "custom-namespace"

	bundler := New(
		WithBundlerTypes([]BundleType{"mock-configurable"}),
		WithConfig(config),
	)
	output, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if output == nil {
		t.Fatal("Make() returned nil output")
	}
}

func TestDefaultBundler_MakeWithAllBundlers(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	// No bundler types specified - should use all registered bundlers
	bundler := New()
	output, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if output == nil {
		t.Fatal("Make() returned nil output")
	}

	// Should have results from all registered bundlers
	if len(output.Results) == 0 {
		t.Error("Expected results from registered bundlers")
	}
}

func TestDefaultBundler_MakeWithEmptyDirectory(t *testing.T) {
	ctx := context.Background()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	// Empty dir should default to current directory
	bundler := New(WithBundlerTypes([]BundleType{"mock"}))
	output, err := bundler.Make(ctx, rec, "")
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if output.OutputDir != "." {
		t.Errorf("Expected output dir to be '.', got %s", output.OutputDir)
	}
}

func TestDefaultBundler_MakeWithNoBundlers(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "cluster",
						Data: map[string]measurement.Reading{
							"version": measurement.Str("1.28.0"),
						},
					},
				},
			},
		},
	}

	// Specify non-existent bundler type
	bundler := New(WithBundlerTypes([]BundleType{"non-existent"}))
	_, err := bundler.Make(ctx, rec, tmpDir)
	if err == nil {
		t.Error("Expected error when no bundlers are selected")
	}
}

func TestBundleOutput_Summary(t *testing.T) {
	output := &BundleOutput{
		TotalFiles: 5,
		TotalSize:  1024,
		Results: []*BundleResult{
			{Success: true},
			{Success: true},
			{Success: false},
		},
	}

	summary := output.Summary()
	if summary == "" {
		t.Error("Summary() returned empty string")
	}
}

func TestBundleOutput_HasErrors(t *testing.T) {
	tests := []struct {
		name   string
		output *BundleOutput
		want   bool
	}{
		{
			name: "no errors",
			output: &BundleOutput{
				Errors: []BundleError{},
			},
			want: false,
		},
		{
			name: "with errors",
			output: &BundleOutput{
				Errors: []BundleError{
					{BundlerType: "test", Error: "test error"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.output.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBundleResult_AddFile(t *testing.T) {
	result := NewBundleResult("test")
	result.AddFile("/path/to/file", 100)

	if len(result.Files) != 1 {
		t.Errorf("AddFile() did not add file, got %d files", len(result.Files))
	}

	if result.Size != 100 {
		t.Errorf("AddFile() size = %d, want 100", result.Size)
	}
}

func TestValidateRecipeStructure(t *testing.T) {
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
			recipe: &recipe.Recipe{
				Measurements: []*measurement.Measurement{
					{
						Type: measurement.TypeK8s,
						Subtypes: []measurement.Subtype{
							{
								Name: "cluster",
								Data: map[string]measurement.Reading{
									"version": measurement.Str("1.28.0"),
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecipeStructure(tt.recipe)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecipeStructure() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
