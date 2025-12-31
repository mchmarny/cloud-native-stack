package gpuoperator

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// ScriptData represents data for generating installation scripts and documentation.
type ScriptData struct {
	Timestamp        string
	Namespace        string
	HelmRepository   string
	HelmChart        string
	HelmChartVersion string
	K8sVersion       string
	GPUType          string
	DriverVersion    string
	MIGStrategy      string
	EnableGDS        bool
	EnableCDI        bool
	Request          *recipe.Query
	Version          string
	RecipeVersion    string
}

// GenerateScriptData creates script data from a recipe and config.
func GenerateScriptData(recipe *recipe.Recipe, config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Namespace:      common.GetConfigValue(config, "namespace", "gpu-operator"),
		HelmRepository: common.GetConfigValue(config, "helm_repository", "https://helm.ngc.nvidia.com/nvidia"),
		HelmChart:      "nvidia/gpu-operator",
		Request:        recipe.Request,
		Version:        common.GetBundlerVersion(config),
		RecipeVersion:  common.GetRecipeBundlerVersion(recipe.Metadata),
	}

	// Extract chart version from config or recipe
	if val, ok := config["helm_chart_version"]; ok && val != "" {
		data.HelmChartVersion = val
	}

	// Extract driver version
	if val, ok := config["driver_version"]; ok && val != "" {
		data.DriverVersion = val
	}

	// Extract MIG strategy
	if val, ok := config["mig_strategy"]; ok && val != "" {
		data.MIGStrategy = val
	}

	// Extract GDS enabled flag
	if val, ok := config["enable_gds"]; ok {
		data.EnableGDS = val == "true" || val == "1"
	}

	// Extract CDI enabled flag
	if val, ok := config["enable_cdi"]; ok {
		data.EnableCDI = val == "true" || val == "1"
	}

	// Extract GPU type from request
	if recipe.Request != nil {
		data.GPUType = recipe.Request.GPU.String()
		if recipe.Request.K8s != nil {
			data.K8sVersion = recipe.Request.K8s.String()
		}
	}

	return data
}

// ToMap converts ScriptData to a map for template rendering.
