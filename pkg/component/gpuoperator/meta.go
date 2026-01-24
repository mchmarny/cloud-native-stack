package gpuoperator

import (
	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
)

// BundleMetadata contains metadata used for README and manifest template rendering.
// This data complements the Helm values map with deployment-specific information
// such as namespace, Helm repository URL, and version tracking.
type BundleMetadata struct {
	Namespace        string
	HelmRepository   string
	HelmChart        string
	HelmChartVersion string
	Version          string
	RecipeVersion    string
}

// GenerateBundleMetadata creates bundle metadata from config map.
func GenerateBundleMetadata(config map[string]string) *BundleMetadata {
	data := &BundleMetadata{
		Namespace:        common.GetConfigValue(config, "namespace", "gpu-operator"),
		HelmRepository:   common.GetConfigValue(config, "helm_repository", "https://helm.ngc.nvidia.com/nvidia"),
		HelmChart:        "nvidia/gpu-operator",
		HelmChartVersion: common.GetConfigValue(config, "helm_chart_version", ""),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(config),
	}

	return data
}
