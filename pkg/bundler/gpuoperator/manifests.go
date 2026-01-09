package gpuoperator

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
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

	// Convert helm values to manifest data - extract Value from ConfigValue
	if dv, ok := helmValues.DriverVersion.Value.(string); ok {
		data.DriverVersion = dv
	}
	if okm, ok := helmValues.UseOpenKernelModule.Value.(bool); ok {
		data.UseOpenKernelModule = okm
	}
	if ms, ok := helmValues.MIGStrategy.Value.(string); ok {
		data.MIGStrategy = ms
	}
	if gds, ok := helmValues.EnableGDS.Value.(bool); ok {
		data.EnableGDS = gds
	}
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

// ToMap converts ManifestData to a map for template rendering.
