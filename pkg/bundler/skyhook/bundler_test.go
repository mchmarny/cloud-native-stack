package skyhook

import (
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

func TestBundler_Make(t *testing.T) {
	harness := internal.NewTestHarness(t, "skyhook").
		WithExpectedFiles([]string{
			"values.yaml",
			"manifests/skyhook.yaml",
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
					"skyhook-operator": "v0.1.0",
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
	expectedImageVersions := map[string]string{
		"skyhook_operator_version": "v0.1.0",
		"skyhook_agent_version":    "v0.1.0",
		"kube_rbac_proxy_version":  "v0.18.1",
		"tuning_version":           "v1.0.0",
		"operator_registry":        "nvcr.io/nvidia",
	}

	for key, expected := range expectedImageVersions {
		if got := configMap[key]; got != expected {
			t.Errorf("configMap[%s] = %s, want %s", key, got, expected)
		}
	}

	// Verify config settings are extracted
	expectedConfigs := map[string]string{
		"runtime_required":            "true",
		"interruption_budget_percent": "100",
		"tuning_interrupt_type":       "reboot",
		"manager_cpu_limit":           "500m",
		"manager_memory_limit":        "128Mi",
		"manager_cpu_request":         "10m",
		"manager_memory_request":      "64Mi",
		"node_selector":               "nvidia.com/gpu.product",
		"node_selector_values":        "H100,GB200",
		"toleration_key":              "nvidia.com/gpu",
		"toleration_value":            "true",
	}

	for key, expected := range expectedConfigs {
		if got := configMap[key]; got != expected {
			t.Errorf("configMap[%s] = %s, want %s", key, got, expected)
		}
	}
}

func TestGenerateHelmValues(t *testing.T) {
	rec := createTestRecipe().Build()
	configMap := map[string]string{
		"skyhook_operator_version": "v0.1.0",
		"kube_rbac_proxy_version":  "v0.18.1",
		"operator_registry":        "nvcr.io/nvidia",
	}

	values := GenerateHelmValues(rec, configMap)

	if values == nil {
		t.Fatal("GenerateHelmValues() returned nil")
	}

	if values.SkyhookOperatorVersion.Value != "v0.1.0" {
		t.Errorf("SkyhookOperatorVersion = %s, want v0.1.0", values.SkyhookOperatorVersion.Value)
	}

	if values.OperatorRegistry.Value != "nvcr.io/nvidia" {
		t.Errorf("OperatorRegistry = %s, want nvcr.io/nvidia", values.OperatorRegistry.Value)
	}
}

func TestGenerateSkyhookCRData(t *testing.T) {
	rec := createTestRecipe().Build()
	configMap := map[string]string{
		"runtime_required":            "true",
		"interruption_budget_percent": "100",
		"tuning_interrupt_type":       "reboot",
	}

	crData := GenerateSkyhookCRData(rec, configMap)

	if crData == nil {
		t.Fatal("GenerateSkyhookCRData() returned nil")
	}

	if crData.RuntimeRequired.Value != "true" {
		t.Errorf("RuntimeRequired = %s, want true", crData.RuntimeRequired.Value)
	}

	if crData.InterruptionBudgetPercent.Value != "100" {
		t.Errorf("InterruptionBudgetPercent = %s, want 100", crData.InterruptionBudgetPercent.Value)
	}

	// Verify GRUB config is built
	if len(crData.GrubConfig) == 0 {
		t.Error("GrubConfig should not be empty")
	}

	// Verify sysctl config is built
	if len(crData.SysctlConfig) == 0 {
		t.Error("SysctlConfig should not be empty")
	}

	// Verify containerd service config is built
	if len(crData.ContainerdServiceConfig) == 0 {
		t.Error("ContainerdServiceConfig should not be empty")
	}
}

func TestGenerateScriptData(t *testing.T) {
	rec := createTestRecipe().Build()
	config := map[string]string{
		"helm_chart_repo": "https://test.repo",
	}

	scriptData := GenerateScriptData(rec, config)

	if scriptData == nil {
		t.Fatal("GenerateScriptData() returned nil")
	}

	if scriptData.HelmChartRepo.Value != "https://test.repo" {
		t.Errorf("HelmChartRepo = %s, want https://test.repo", scriptData.HelmChartRepo.Value)
	}
}

func TestGetTemplate(t *testing.T) {
	internal.TestTemplateGetter(t, GetTemplate, []string{
		"values.yaml",
		"skyhook.yaml",
		"install.sh",
		"uninstall.sh",
		"README.md",
	})
}

// createTestRecipe creates a recipe for testing that matches the actual recipe.yaml structure.
func createTestRecipe() *internal.RecipeBuilder {
	return internal.NewRecipeBuilder().
		WithK8sMeasurement(
			internal.ImageSubtype(map[string]string{
				"skyhook-operator":      "v0.1.0",
				"skyhook-agent":         "v0.1.0",
				"kube-rbac-proxy":       "v0.18.1",
				"nodeos-updater-tuning": "v1.0.0",
			}),
			internal.RegistrySubtype(map[string]string{
				"uri": "nvcr.io/nvidia",
			}),
			createSkyhookConfigSubtype(),
		).
		WithOSMeasurement(
			internal.GrubSubtype(map[string]string{
				"hugepagesz":         "2M",
				"hugepages":          "8192",
				"nokaslr":            "",
				"numa_balancing":     "disable",
				"default_hugepagesz": "64K",
				"iommu.passthrough":  "1",
				"init_on_alloc":      "0",
			}),
			internal.SysctlSubtype(map[string]string{
				"/proc/sys/kernel/threads-max": "4194304",
				"/proc/sys/vm/max_map_count":   "2147483647",
				"/proc/sys/fs/file-max":        "9223372036854775807",
			}),
		).
		WithSystemDMeasurement(
			internal.ServiceSubtype("containerd.service", map[string]string{
				"LimitSTACK": "infinity",
			}),
		)
}

// createSkyhookConfigSubtype creates a skyhook-config subtype for testing.
func createSkyhookConfigSubtype() measurement.Subtype {
	return measurement.Subtype{
		Name: "skyhook-config",
		Data: map[string]measurement.Reading{
			"runtime_required":            measurement.Str("true"),
			"interruption_budget_percent": measurement.Str("100"),
			"tuning_interrupt_type":       measurement.Str("reboot"),
			"manager_cpu_limit":           measurement.Str("500m"),
			"manager_memory_limit":        measurement.Str("128Mi"),
			"manager_cpu_request":         measurement.Str("10m"),
			"manager_memory_request":      measurement.Str("64Mi"),
			"node_selector":               measurement.Str("nvidia.com/gpu.product"),
			"node_selector_values":        measurement.Str("H100,GB200"),
			"toleration_key":              measurement.Str("nvidia.com/gpu"),
			"toleration_value":            measurement.Str("true"),
		},
	}
}
