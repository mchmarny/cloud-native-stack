package nvsentinel

import (
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

func TestBundler_Make(t *testing.T) {
	harness := internal.NewTestHarness(t, "nvsentinel").
		WithExpectedFiles([]string{
			"values.yaml",
			"scripts/install.sh",
			"scripts/uninstall.sh",
			"README.md",
			"checksums.txt",
		}).
		WithRecipeBuilder(createTestRecipe)

	b := NewBundler(config.NewConfig())
	harness.TestMake(b)
}

func TestBundler_Validate(t *testing.T) {
	tests := []struct {
		name    string
		recipe  *internal.RecipeBuilder
		wantErr bool
	}{
		{
			name: "valid recipe",
			recipe: internal.NewRecipeBuilder().
				WithK8sMeasurement(internal.ImageSubtype(map[string]string{
					"nvsentinel": "v0.6.0",
				})),
			wantErr: false,
		},
		{
			name: "missing K8s measurements",
			recipe: internal.NewRecipeBuilder().
				WithGPUMeasurement(internal.SMISubtype(map[string]string{
					"driver-version": "580.82.07",
				})),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := tt.recipe.Build()
			err := rec.ValidateMeasurementExists(measurement.TypeK8s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMeasurementExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBundler_Configure(t *testing.T) {
	cfg := config.NewConfig()
	b := NewBundler(cfg)

	// Verify default values
	if b.Config.Namespace() != "default" {
		t.Errorf("Configure() namespace = %s, want default", b.Config.Namespace())
	}
}

func TestBundler_buildConfigMap(t *testing.T) {
	b := NewBundler(config.NewConfig())
	rec := createTestRecipe().Build()

	configMap := b.buildConfigMap(rec)

	// Verify namespace is set
	if configMap["namespace"] != Name {
		t.Errorf("namespace = %s, want %s", configMap["namespace"], Name)
	}

	// Verify K8s image versions are extracted
	if got := configMap["nvsentinel_version"]; got != "v0.6.0" {
		t.Errorf("configMap[nvsentinel_version] = %s, want v0.6.0", got)
	}
}

func TestGenerateHelmValues(t *testing.T) {
	rec := createTestRecipe().Build()
	config := map[string]string{
		"namespace":          Name,
		"nvsentinel_version": "v0.6.0",
	}

	values := GenerateHelmValues(rec, config)

	// Verify basic fields
	if values.Namespace != Name {
		t.Errorf("Namespace = %s, want %s", values.Namespace, Name)
	}

	if values.NVSentinelVersion.Value != "v0.6.0" {
		t.Errorf("NVSentinelVersion = %s, want v0.6.0", values.NVSentinelVersion.Value)
	}
}

func TestGenerateScriptData(t *testing.T) {
	rec := createTestRecipe().Build()
	config := map[string]string{
		"namespace":          Name,
		"nvsentinel_version": "v0.6.0",
	}

	data := GenerateScriptData(rec, config)

	// Verify basic fields
	if data.Namespace != Name {
		t.Errorf("Namespace = %s, want %s", data.Namespace, Name)
	}

	if data.NVSentinelVersion.Value != "v0.6.0" {
		t.Errorf("NVSentinelVersion = %s, want v0.6.0", data.NVSentinelVersion.Value)
	}

	expectedRepo := "oci://ghcr.io/nvidia/nvsentinel"
	if data.HelmChartRepo.Value != expectedRepo {
		t.Errorf("HelmChartRepo = %s, want %s", data.HelmChartRepo.Value, expectedRepo)
	}

	expectedRelease := "nvsentinel"
	if data.HelmReleaseName.Value != expectedRelease {
		t.Errorf("HelmReleaseName = %s, want %s", data.HelmReleaseName.Value, expectedRelease)
	}
}

// createTestRecipe creates a test recipe with sample measurements.
func createTestRecipe() *internal.RecipeBuilder {
	return internal.NewRecipeBuilder().
		WithK8sMeasurement(
			internal.ImageSubtype(map[string]string{
				"nvsentinel": "v0.6.0",
			}),
			internal.RegistrySubtype(map[string]string{
				"uri": "ghcr.io/nvidia",
			}),
		)
}
