package nvsentinel

import (
	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
)

// BundleMetadata contains metadata used for README and manifest template rendering.
// This data complements the Helm values map with deployment-specific information.
type BundleMetadata struct {
	Version           string
	RecipeVersion     string
	Namespace         string
	HelmChartRepo     string
	HelmReleaseName   string
	NVSentinelVersion string
}

// GenerateBundleMetadata creates bundle metadata from config map.
func GenerateBundleMetadata(config map[string]string) *BundleMetadata {
	data := &BundleMetadata{
		Version:           common.GetBundlerVersion(config),
		RecipeVersion:     common.GetRecipeBundlerVersion(config),
		Namespace:         common.GetConfigValue(config, "namespace", Name),
		HelmChartRepo:     common.GetConfigValue(config, "helm_repository", "oci://ghcr.io/nvidia/nvsentinel"),
		HelmReleaseName:   common.GetConfigValue(config, "helm_release_name", "nvsentinel"),
		NVSentinelVersion: common.GetConfigValue(config, "helm_chart_version", "v0.6.0"),
	}

	return data
}
