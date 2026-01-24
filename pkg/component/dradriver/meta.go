package dradriver

import (
	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
)

// BundleMetadata contains metadata used for README and manifest template rendering.
// This data complements the Helm values map with deployment-specific information.
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
		Namespace:        common.GetConfigValue(config, "namespace", "nvidia-dra-driver"),
		HelmRepository:   common.GetConfigValue(config, "helm_repository", "https://helm.ngc.nvidia.com/nvidia"),
		HelmChart:        "nvidia/k8s-dra-driver",
		HelmChartVersion: common.GetConfigValue(config, "helm_chart_version", ""),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(config),
	}

	return data
}
