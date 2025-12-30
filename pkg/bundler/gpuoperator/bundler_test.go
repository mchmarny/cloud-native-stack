package gpuoperator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/common"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const testNamespace = "test-ns"

func TestBundler_Make(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	rec := createTestRecipe()
	b := NewBundler(config.NewConfig())

	result, err := b.Make(ctx, rec, tmpDir)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if result == nil {
		t.Fatal("Make() returned nil result")
		return
	}

	if !result.Success {
		t.Error("Make() should succeed")
	}

	if len(result.Files) == 0 {
		t.Error("Make() produced no files")
	}

	// Verify bundle directory structure
	bundleDir := filepath.Join(tmpDir, "gpu-operator")
	if _, err := os.Stat(bundleDir); os.IsNotExist(err) {
		t.Error("Make() did not create gpu-operator directory")
	}

	// Verify key files exist
	expectedFiles := []string{
		"values.yaml",
		"manifests/clusterpolicy.yaml",
		"scripts/install.sh",
		"scripts/uninstall.sh",
		"README.md",
		"checksums.txt",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(bundleDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s not found", file)
		}
	}
}

func TestBundler_Validate(t *testing.T) {
	tests := []struct {
		name    string
		recipe  *recipe.Recipe
		wantErr bool
	}{
		{
			name:    "valid recipe",
			recipe:  createTestRecipe(),
			wantErr: false,
		},
		{
			name: "missing K8s measurements",
			recipe: &recipe.Recipe{
				Measurements: []*measurement.Measurement{
					{
						Type: measurement.TypeGPU,
						Subtypes: []measurement.Subtype{
							{Name: "device", Data: map[string]measurement.Reading{}},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.recipe.ValidateMeasurementExists(measurement.TypeK8s)
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
	if b.config.Namespace() != "default" {
		t.Errorf("Configure() namespace = %s, want default", b.config.Namespace())
	}
}

func TestGenerateHelmValues(t *testing.T) {
	rec := createTestRecipe()
	config := make(map[string]string)
	config["namespace"] = testNamespace

	values := GenerateHelmValues(rec, config)

	if values == nil {
		t.Fatal("GenerateHelmValues() returned nil")
		return
	}

	if values.Namespace != testNamespace {
		t.Errorf("Namespace = %s, want %s", values.Namespace, testNamespace)
	}

	// Verify extracted values from recipe match expected structure
	if v, ok := values.GPUOperatorVersion.Value.(string); !ok || v != "v25.3.3" {
		t.Errorf("GPUOperatorVersion = %v, want v25.3.3", values.GPUOperatorVersion.Value)
	}

	if v, ok := values.DriverVersion.Value.(string); !ok || v != "580.82.07" {
		t.Errorf("DriverVersion = %v, want 580.82.07", values.DriverVersion.Value)
	}

	if v, ok := values.NvidiaContainerToolkitVersion.Value.(string); !ok || v != "v1.17.8" {
		t.Errorf("NvidiaContainerToolkitVersion = %v, want v1.17.8", values.NvidiaContainerToolkitVersion.Value)
	}

	if v, ok := values.DevicePluginVersion.Value.(string); !ok || v != "v0.17.4" {
		t.Errorf("DevicePluginVersion = %v, want v0.17.4", values.DevicePluginVersion.Value)
	}

	if v, ok := values.DCGMVersion.Value.(string); !ok || v != "4.3.1-1" {
		t.Errorf("DCGMVersion = %v, want 4.3.1-1", values.DCGMVersion.Value)
	}

	if v, ok := values.DCGMExporterVersion.Value.(string); !ok || v != "4.3.1" {
		t.Errorf("DCGMExporterVersion = %v, want 4.3.1", values.DCGMExporterVersion.Value)
	}

	if v, ok := values.UseOpenKernelModule.Value.(bool); !ok || !v {
		t.Error("UseOpenKernelModule = false, want true")
	}

	if v, ok := values.EnableGDS.Value.(bool); !ok || !v {
		t.Error("EnableGDS = false, want true (from RDMA)")
	}

	if v, ok := values.MIGStrategy.Value.(string); !ok || v != "single" {
		t.Errorf("MIGStrategy = %v, want single (mig=false)", values.MIGStrategy.Value)
	}

	if err := values.Validate(); err != nil {
		t.Errorf("HelmValues.Validate() error = %v", err)
	}
}

func TestGenerateManifestData(t *testing.T) {
	rec := createTestRecipe()
	config := make(map[string]string)
	config["namespace"] = testNamespace

	data := GenerateManifestData(rec, config)

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
	if data.DriverVersion != "580.82.07" {
		t.Errorf("DriverVersion = %s, want 580.82.07", data.DriverVersion)
	}

	if !data.UseOpenKernelModule {
		t.Error("UseOpenKernelModule = false, want true")
	}

	if !data.EnableGDS {
		t.Error("EnableGDS = false, want true")
	}
}

func TestGenerateScriptData(t *testing.T) {
	rec := createTestRecipe()
	config := make(map[string]string)
	config["helm_repository"] = "https://test.repo"

	data := GenerateScriptData(rec, config)

	if data == nil {
		t.Fatal("GenerateScriptData() returned nil")
		return
	}

	if data.HelmRepository != "https://test.repo" {
		t.Errorf("HelmRepository = %s, want https://test.repo", data.HelmRepository)
	}
}

func TestGetTemplate(t *testing.T) {
	tests := []string{
		"values.yaml",
		"clusterpolicy",
		"install.sh",
		"uninstall.sh",
		"README.md",
	}

	for _, name := range tests {
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
	_, ok := GetTemplate("nonexistent")
	if ok {
		t.Error("GetTemplate() should return false for non-existent template")
	}
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
				DriverRegistry: common.ValueWithContext{Value: "nvcr.io/nvidia"},
				MIGStrategy:    common.ValueWithContext{Value: "single"},
			},
			wantErr: false,
		},
		{
			name: "empty namespace",
			values: &HelmValues{
				Namespace:      "",
				DriverRegistry: common.ValueWithContext{Value: "nvcr.io/nvidia"},
				MIGStrategy:    common.ValueWithContext{Value: "single"},
			},
			wantErr: true,
		},
		{
			name: "invalid MIG strategy",
			values: &HelmValues{
				Namespace:      "test",
				DriverRegistry: common.ValueWithContext{Value: "nvcr.io/nvidia"},
				MIGStrategy:    common.ValueWithContext{Value: "invalid"},
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
func createTestRecipe() *recipe.Recipe {
	return &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "image",
						Data: map[string]measurement.Reading{
							"gpu-operator":      measurement.Str("v25.3.3"),
							"driver":            measurement.Str("580.82.07"),
							"container-toolkit": measurement.Str("v1.17.8"),
							"k8s-device-plugin": measurement.Str("v0.17.4"),
							"dcgm":              measurement.Str("4.3.1-1"),
							"dcgm-exporter":     measurement.Str("4.3.1"),
						},
					},
					{
						Name: "config",
						Data: map[string]measurement.Reading{
							"cdi":                 measurement.Bool(true),
							"mig":                 measurement.Bool(false),
							"rdma":                measurement.Bool(true),
							"useOpenKernelModule": measurement.Bool(true),
						},
					},
				},
			},
			{
				Type: measurement.TypeGPU,
				Subtypes: []measurement.Subtype{
					{
						Name: "smi",
						Data: map[string]measurement.Reading{
							"driver-version": measurement.Str("580.82.07"),
							"cuda-version":   measurement.Str("13.1"),
						},
					},
				},
			},
		},
	}
}
