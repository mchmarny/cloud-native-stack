package skyhook

import (
	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
)

// BundleMetadata represents metadata for bundle generation (README, manifests).
// This struct provides deployment metadata used in README templates and manifest generation.
type BundleMetadata struct {
	Version          string
	RecipeVersion    string
	Namespace        string
	HelmChartRepo    string
	HelmChartName    string
	HelmReleaseName  string
	OperatorRegistry string
}

// GenerateBundleMetadata creates bundle metadata from config map.
func GenerateBundleMetadata(config map[string]string) *BundleMetadata {
	data := &BundleMetadata{
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(config),
		Namespace:        common.GetConfigValue(config, "namespace", Name),
		HelmChartRepo:    common.GetConfigValue(config, "helm_repository", "https://nvidia.github.io/skyhook"),
		HelmChartName:    common.GetConfigValue(config, "helm_chart_name", "skyhook"),
		HelmReleaseName:  common.GetConfigValue(config, "helm_release_name", "skyhook"),
		OperatorRegistry: common.GetConfigValue(config, "operator_registry", "nvcr.io/nvidia"),
	}

	return data
}
