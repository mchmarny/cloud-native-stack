package certmanager

import (
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
)

const trueValue = "true"

func TestBundler_Make(t *testing.T) {
	harness := internal.NewTestHarness(t, "cert-manager").
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

	// Verify cert-manager version is extracted (if present in recipe)
	if val, ok := configMap["cert_manager_version"]; ok {
		if val != "v1.19.1" {
			t.Errorf("cert_manager_version = %s, want v1.19.1", val)
		}
	}
}

func TestGenerateHelmValues(t *testing.T) {
	rec := createTestRecipe().Build()
	configMap := map[string]string{
		"namespace":            Name,
		"cert_manager_version": "v1.19.1",
		"install_crds":         "true",
		"enable_webhook":       "true",
		"replica_count":        "1",
	}

	helmValues := GenerateHelmValues(rec, configMap, nil)

	// Verify basic fields
	if helmValues.Namespace != Name {
		t.Errorf("Namespace = %s, want %s", helmValues.Namespace, Name)
	}

	if helmValues.InstallCRDs != trueValue {
		t.Errorf("InstallCRDs = %s, want true", helmValues.InstallCRDs)
	}

	if helmValues.EnablePrometheus != trueValue {
		t.Errorf("EnablePrometheus = %v, want true", helmValues.EnablePrometheus)
	}

	// Verify controller resources
	if helmValues.ControllerCPURequest != "50m" {
		t.Errorf("ControllerCPURequest = %v, want 50m", helmValues.ControllerCPURequest)
	}

	if helmValues.ControllerMemoryRequest != "90Mi" {
		t.Errorf("ControllerMemoryRequest = %v, want 90Mi", helmValues.ControllerMemoryRequest)
	}

	// Verify scheduling
	if helmValues.TolerationKey != "dedicated" {
		t.Errorf("TolerationKey = %v, want dedicated", helmValues.TolerationKey)
	}

	if helmValues.NodeSelectorValue != "system-cpu" {
		t.Errorf("NodeSelectorValue = %v, want system-cpu", helmValues.NodeSelectorValue)
	}
}

func TestGenerateScriptData(t *testing.T) {
	rec := createTestRecipe().Build()
	configMap := map[string]string{
		"namespace":            Name,
		"helm_repository":      "https://charts.jetstack.io",
		"cert_manager_version": "v1.19.1",
		"install_crds":         "true",
	}

	scriptData := GenerateScriptData(rec, configMap)

	// Verify basic fields
	if scriptData.Namespace != Name {
		t.Errorf("Namespace = %s, want %s", scriptData.Namespace, Name)
	}

	if scriptData.HelmRepository != "https://charts.jetstack.io" {
		t.Errorf("HelmRepository = %s, want https://charts.jetstack.io", scriptData.HelmRepository)
	}

	if scriptData.HelmChart != "jetstack/cert-manager" {
		t.Errorf("HelmChart = %s, want jetstack/cert-manager", scriptData.HelmChart)
	}

	if !scriptData.InstallCRDs {
		t.Errorf("InstallCRDs = %v, want true", scriptData.InstallCRDs)
	}
}

func createTestRecipe() *internal.RecipeBuilder {
	return internal.NewRecipeBuilder().
		WithK8sMeasurement(
			internal.ImageSubtype(map[string]string{
				"cert-manager": "v1.19.1",
			}),
			internal.ConfigSubtype(map[string]interface{}{
				"install-crds":   "true",
				"enable-webhook": "true",
				"replica-count":  "1",
			}),
		)
}
