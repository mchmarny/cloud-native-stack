package gpuoperator

import (
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

const (
	testNamespace     = "test-ns"
	testDriverVersion = "580.82.07"
)

func TestBundler_Make(t *testing.T) {
	harness := internal.NewTestHarness(t, "gpu-operator").
		WithExpectedFiles([]string{
			"values.yaml",
			"manifests/clusterpolicy.yaml",
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
					"gpu-operator": "v25.3.3",
				})),
			wantErr: false,
		},
		{
			name: "missing K8s measurements",
			recipe: internal.NewRecipeBuilder().
				WithGPUMeasurement(internal.SMISubtype(map[string]string{
					"driver-version": testDriverVersion,
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
	// Note: Config is now immutable. We cannot assign fields directly.
	// Instead, bundlers should receive pre-configured Config instances.

	// Create a test config for verification purposes
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
	expectedImageVersions := map[string]string{
		"gpu_operator_version":      "v25.3.3",
		"driver_version":            testDriverVersion,
		"container_toolkit_version": "v1.17.8",
		"device_plugin_version":     "v0.17.4",
		"dcgm_version":              "4.3.1-1",
		"dcgm_exporter_version":     "4.3.1",
		"driver_registry":           "nvcr.io/nvidia",
	}

	for key, expected := range expectedImageVersions {
		if got := configMap[key]; got != expected {
			t.Errorf("%s = %s, want %s", key, got, expected)
		}
	}

	// Verify K8s config flags are extracted
	expectedConfigFlags := map[string]string{
		"enable_cdi":             "true",
		"enable_mig":             "false",
		"enable_gds":             "true",
		"use_open_kernel_module": "true",
	}

	for key, expected := range expectedConfigFlags {
		if got := configMap[key]; got != expected {
			t.Errorf("%s = %s, want %s", key, got, expected)
		}
	}

	// Verify MIG strategy is NOT set to "mixed" when MIG is false
	if val, exists := configMap["mig_strategy"]; exists {
		t.Errorf("mig_strategy should not be set when MIG is disabled, got: %s", val)
	}
}

func TestBundler_buildConfigMap_MIGEnabled(t *testing.T) {
	b := NewBundler(config.NewConfig())

	// Create recipe with MIG enabled
	rec := internal.NewRecipeBuilder().
		WithK8sMeasurement(
			internal.ImageSubtype(map[string]string{
				"gpu-operator": "v25.3.3",
			}),
			internal.ConfigSubtype(map[string]interface{}{
				"mig": true,
			}),
		).Build()

	configMap := b.buildConfigMap(rec)

	// Verify MIG is enabled
	if got := configMap["enable_mig"]; got != strTrue {
		t.Errorf("enable_mig = %s, want true", got)
	}

	// Verify MIG strategy is set to "mixed" when MIG is enabled
	if got := configMap["mig_strategy"]; got != strMixed {
		t.Errorf("mig_strategy = %s, want mixed", got)
	}
}

func TestBundler_buildConfigMap_GPUDriverVersion(t *testing.T) {
	b := NewBundler(config.NewConfig())

	// Create recipe with driver version only in GPU measurements (not K8s)
	rec := internal.NewRecipeBuilder().
		WithK8sMeasurement(
			internal.ImageSubtype(map[string]string{
				"gpu-operator": "v25.3.3",
			}),
		).
		WithGPUMeasurement(
			internal.SMISubtype(map[string]string{
				"driver-version": "550.90.07",
			}),
		).Build()

	configMap := b.buildConfigMap(rec)

	// Verify driver version from GPU measurement is extracted
	if got := configMap["driver_version"]; got != "550.90.07" {
		t.Errorf("driver_version = %s, want 550.90.07", got)
	}
}

func TestBundler_buildConfigMap_K8sDriverVersionTakesPrecedence(t *testing.T) {
	b := NewBundler(config.NewConfig())

	// Create recipe with driver version in both K8s and GPU measurements
	rec := internal.NewRecipeBuilder().
		WithK8sMeasurement(
			internal.ImageSubtype(map[string]string{
				"gpu-operator": "v25.3.3",
				"driver":       testDriverVersion,
			}),
		).
		WithGPUMeasurement(
			internal.SMISubtype(map[string]string{
				"driver-version": "550.90.07",
			}),
		).Build()

	configMap := b.buildConfigMap(rec)

	// Verify K8s driver version takes precedence
	if got := configMap["driver_version"]; got != testDriverVersion {
		t.Errorf("driver_version = %s, want %s (K8s version should take precedence)", got, testDriverVersion)
	}
}

func TestGenerateHelmValues(t *testing.T) {
	rec := createTestRecipe().Build()
	config := make(map[string]string)
	config["namespace"] = testNamespace

	values := GenerateHelmValues(rec, config, nil)

	if values == nil {
		t.Fatal("GenerateHelmValues() returned nil")
		return
	}

	if values.Namespace != testNamespace {
		t.Errorf("Namespace = %s, want %s", values.Namespace, testNamespace)
	}

	// Verify extracted values from recipe match expected structure
	if values.GPUOperatorVersion != "v25.3.3" {
		t.Errorf("GPUOperatorVersion = %s, want v25.3.3", values.GPUOperatorVersion)
	}
	if values.DriverVersion != testDriverVersion {
		t.Errorf("DriverVersion = %s, want %s", values.DriverVersion, testDriverVersion)
	}
	if values.NvidiaContainerToolkitVersion != "v1.17.8" {
		t.Errorf("NvidiaContainerToolkitVersion = %s, want v1.17.8", values.NvidiaContainerToolkitVersion)
	}
	if values.DevicePluginVersion != "v0.17.4" {
		t.Errorf("DevicePluginVersion = %s, want v0.17.4", values.DevicePluginVersion)
	}
	if values.DCGMVersion != "4.3.1-1" {
		t.Errorf("DCGMVersion = %s, want 4.3.1-1", values.DCGMVersion)
	}
	if values.DCGMExporterVersion != "4.3.1" {
		t.Errorf("DCGMExporterVersion = %s, want 4.3.1", values.DCGMExporterVersion)
	}
	if values.UseOpenKernelModule != "true" {
		t.Errorf("UseOpenKernelModule = %s, want true", values.UseOpenKernelModule)
	}
	if values.EnableGDS != "true" {
		t.Errorf("EnableGDS = %s, want true", values.EnableGDS)
	}
	if values.MIGStrategy != "single" {
		t.Errorf("MIGStrategy = %s, want single", values.MIGStrategy)
	}

	if err := values.Validate(); err != nil {
		t.Errorf("HelmValues.Validate() error = %v", err)
	}
}

func TestGenerateManifestData(t *testing.T) {
	rec := createTestRecipe().Build()
	config := make(map[string]string)
	config["namespace"] = testNamespace

	data := GenerateManifestData(rec, config, nil)

	if data == nil {
		t.Fatal("GenerateManifestData() returned nil")
		return
	}

	if data.Namespace != testNamespace {
		t.Errorf("Namespace = %s, want %s", data.Namespace, testNamespace)
	}

	// Verify CDI is extracted from recipe config subtype
	if !data.EnableCDI {
		t.Error("EnableCDI = false, want true (from recipe config)")
	}

	// Verify other values propagated from HelmValues
	if data.DriverVersion != testDriverVersion {
		t.Errorf("DriverVersion = %s, want %s", data.DriverVersion, testDriverVersion)
	}

	if !data.UseOpenKernelModule {
		t.Error("UseOpenKernelModule = false, want true")
	}

	if !data.EnableGDS {
		t.Error("EnableGDS = false, want true")
	}
}

func TestGenerateScriptData(t *testing.T) {
	rec := createTestRecipe().Build()
	config := make(map[string]string)
	config["helm_repository"] = "https://test.repo"

	data := GenerateScriptData(rec, config)

	if data == nil {
		t.Fatal("GenerateScriptData() returned nil")
		return
	}

	internal.AssertConfigValue(t, config, "helm_repository", "https://test.repo")
}

func TestGetTemplate(t *testing.T) {
	internal.TestTemplateGetter(t, GetTemplate, []string{
		"values.yaml",
		"clusterpolicy",
		"install.sh",
		"uninstall.sh",
		"README.md",
	})
}

func TestHelmValues_Validate(t *testing.T) {
	tests := []struct {
		name    string
		values  *HelmValues
		wantErr bool
	}{
		{
			name: "valid values",
			values: &HelmValues{
				Namespace:      "test",
				DriverRegistry: "nvcr.io/nvidia",
				MIGStrategy:    "single",
			},
			wantErr: false,
		},
		{
			name: "empty namespace",
			values: &HelmValues{
				Namespace:      "",
				DriverRegistry: "nvcr.io/nvidia",
				MIGStrategy:    "single",
			},
			wantErr: true,
		},
		{
			name: "invalid MIG strategy",
			values: &HelmValues{
				Namespace:      "test",
				DriverRegistry: "nvcr.io/nvidia",
				MIGStrategy:    "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.values.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// createTestRecipe creates a recipe for testing that matches the actual recipe.yaml structure.
func createTestRecipe() *internal.RecipeBuilder {
	return internal.NewRecipeBuilder().
		WithK8sMeasurement(
			internal.RegistrySubtype(map[string]string{
				"name": "nvcr.io",
				"repo": "nvidia",
				"uri":  "nvcr.io/nvidia",
			}),
			internal.ImageSubtype(map[string]string{
				"gpu-operator":      "v25.3.3",
				"driver":            testDriverVersion,
				"container-toolkit": "v1.17.8",
				"k8s-device-plugin": "v0.17.4",
				"dcgm":              "4.3.1-1",
				"dcgm-exporter":     "4.3.1",
			}),
			internal.ConfigSubtype(map[string]interface{}{
				"cdi":                 true,
				"mig":                 false,
				"rdma":                true,
				"useOpenKernelModule": true,
			}),
		).
		WithGPUMeasurement(
			internal.SMISubtype(map[string]string{
				"driver-version": "580.82.07",
				"cuda-version":   "13.1",
			}),
		)
}
