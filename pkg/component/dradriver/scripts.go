package dradriver

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
)

// ScriptData represents metadata for generating installation scripts.
// This contains information not present in the Helm values map.
type ScriptData struct {
	Timestamp        string
	Namespace        string
	HelmRepository   string
	HelmChart        string
	HelmChartVersion string
	Version          string
	RecipeVersion    string
}

// GenerateScriptDataFromConfig creates script data from config map.
func GenerateScriptDataFromConfig(config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		Namespace:        common.GetConfigValue(config, "namespace", "nvidia-dra-driver"),
		HelmRepository:   common.GetConfigValue(config, "helm_repository", "https://helm.ngc.nvidia.com/nvidia"),
		HelmChart:        "nvidia/k8s-dra-driver",
		HelmChartVersion: common.GetConfigValue(config, "helm_chart_version", ""),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(config),
	}

	return data
}

// ToMap converts ScriptData to a map for template rendering.
