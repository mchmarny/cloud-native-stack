package gpuoperator

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// ManifestData represents data for generating Kubernetes manifests.
type ManifestData struct {
	Timestamp           string
	Namespace           string
	EnableDriver        bool
	DriverVersion       string
	UseOpenKernelModule bool
	MIGStrategy         string
	EnableGDS           bool
	EnableVGPU          bool
	VGPULicenseServer   string
	EnableCDI           bool
	CustomLabels        map[string]string
	CustomAnnotations   map[string]string
	Version             string
	RecipeVersion       string
}

// GenerateManifestData creates manifest data from a recipe and config.
func GenerateManifestData(recipe *recipe.Recipe, config map[string]string, overrides map[string]string) *ManifestData {
	data := &ManifestData{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		Namespace:         common.GetConfigValue(config, "namespace", "gpu-operator"),
		EnableDriver:      true,
		MIGStrategy:       "single",
		EnableGDS:         false,
		EnableVGPU:        false,
		EnableCDI:         false,
		CustomLabels:      common.ExtractCustomLabels(config),
		CustomAnnotations: common.ExtractCustomAnnotations(config),
		Version:           common.GetBundlerVersion(config),
		RecipeVersion:     common.GetRecipeBundlerVersion(recipe.Metadata),
	}

	// Extract values from recipe (similar to HelmValues)
	helmValues := GenerateHelmValues(recipe, config, overrides)

	// Copy helm values to manifest data
	data.DriverVersion = helmValues.DriverVersion
	data.UseOpenKernelModule = helmValues.UseOpenKernelModule == strTrue
	data.MIGStrategy = helmValues.MIGStrategy
	data.EnableGDS = helmValues.EnableGDS == strTrue
	data.CustomLabels = helmValues.CustomLabels

	// Extract CDI setting from K8s config subtype
	for _, m := range recipe.Measurements {
		if m.Type == measurement.TypeK8s {
			for _, st := range m.Subtypes {
				if st.Name == "config" {
					if val, ok := st.Data["cdi"]; ok {
						if b, ok := val.Any().(bool); ok {
							data.EnableCDI = b
						}
					}
				}
			}
		}
	}

	// Apply config-specific manifest settings (overrides)
	if val, ok := config["enable_vgpu"]; ok {
		data.EnableVGPU = val == strTrue
	}
	if val, ok := config["vgpu_license_server"]; ok && val != "" {
		data.VGPULicenseServer = val
	}
	if val, ok := config["enable_cdi"]; ok {
		data.EnableCDI = val == strTrue
	}

	return data
}

// GenerateManifestDataFromValues creates manifest data from HelmValues (for RecipeResult inputs).
func GenerateManifestDataFromValues(helmValues *HelmValues) *ManifestData {
	data := &ManifestData{
		Timestamp:     helmValues.Timestamp,
		Namespace:     helmValues.Namespace,
		EnableDriver:  true,
		MIGStrategy:   "single",
		EnableGDS:     false,
		EnableVGPU:    false,
		EnableCDI:     false,
		CustomLabels:  helmValues.CustomLabels,
		Version:       helmValues.Version,
		RecipeVersion: helmValues.RecipeVersion,
	}

	// Copy helm values to manifest data
	data.DriverVersion = helmValues.DriverVersion
	data.UseOpenKernelModule = helmValues.UseOpenKernelModule == strTrue
	data.MIGStrategy = helmValues.MIGStrategy
	data.EnableGDS = helmValues.EnableGDS == strTrue

	return data
}

// ToMap converts ManifestData to a map for template rendering.
