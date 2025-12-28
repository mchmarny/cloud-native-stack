package bundler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/bundle"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

var (
	testReg = NewRegistry(config.NewConfig())
)

func init() {
	testReg.Register("mock", &mockBundler{})
	testReg.Register("mock-configurable", &mockConfigurableBundler{})
	testReg.Register("mock-fail", &mockBundler{shouldFail: true})
}

// mockBundler for testing.
type mockBundler struct {
	shouldFail bool
}

func (m *mockBundler) Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*bundle.Result, error) {
	result := bundle.NewResult("mock")
	if m.shouldFail {
		return result, errors.New(errors.ErrCodeInternal, "mock bundler failed")
	}
	result.AddFile("test.txt", 100)
	result.MarkSuccess()
	return result, nil
}

// mockConfigurableBundler for testing configuration.
type mockConfigurableBundler struct {
	config *config.Config
}

func (m *mockConfigurableBundler) Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*bundle.Result, error) {
	result := bundle.NewResult("mock-configurable")
	result.AddFile("test.txt", 100)
	result.MarkSuccess()
	return result, nil
}

func (m *mockConfigurableBundler) Configure(config *config.Config) error {
	if config == nil {
		return errors.New(errors.ErrCodeInvalidRequest, "config cannot be nil")
	}
	m.config = config
	return nil
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

	bundler := New(
		WithRegistry(testReg),
		WithBundlerTypes([]bundle.Type{"mock"}),
	)
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

	config := config.NewConfig()
	config.Namespace = "test-namespace"

	bundler := New(WithRegistry(testReg), WithBundlerTypes([]bundle.Type{"mock"}),
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

	bundler := New(
		WithRegistry(testReg),
		WithBundlerTypes([]bundle.Type{"mock"}),
	)
	_, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Make() did not create output directory")
	}
}

func TestDefaultBundler_MakeWithMultipleBundlers(t *testing.T) {
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

	// Bundlers execute in parallel
	bundler := New(
		WithRegistry(testReg),
	)
	output, err := bundler.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if output == nil {
		t.Fatal("Make() returned nil output")
	}

	// Should have results from both bundlers
	if len(output.Results) != testReg.Count() {
		t.Errorf("Expected %d results, got %d", testReg.Count(), len(output.Results))
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
		WithRegistry(testReg),
		WithBundlerTypes([]bundle.Type{"mock-fail", "mock"}),
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
		WithRegistry(testReg),
		WithBundlerTypes([]bundle.Type{"mock-fail", "mock"}),
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

	config := config.NewConfig()
	config.Namespace = "custom-namespace"

	bundler := New(WithRegistry(testReg), WithBundlerTypes([]bundle.Type{"mock-configurable"}),
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
	bundler := New(
		WithRegistry(testReg),
		WithBundlerTypes([]bundle.Type{"mock"}),
	)
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
	bundler := New(WithBundlerTypes([]bundle.Type{"non-existent"}))
	_, err := bundler.Make(ctx, rec, tmpDir)
	if err == nil {
		t.Error("Expected error when no bundlers are selected")
	}
}

func TestBundleOutput_Summary(t *testing.T) {
	output := &bundle.Output{
		TotalFiles: 5,
		TotalSize:  1024,
		Results: []*bundle.Result{
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
		output *bundle.Output
		want   bool
	}{
		{
			name: "no errors",
			output: &bundle.Output{
				Errors: []bundle.BundleError{},
			},
			want: false,
		},
		{
			name: "with errors",
			output: &bundle.Output{
				Errors: []bundle.BundleError{
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
	result := bundle.NewResult("test")
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
			err := tt.recipe.ValidateStructure()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecipeStructure() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
