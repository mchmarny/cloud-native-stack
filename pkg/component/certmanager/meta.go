package certmanager

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
	InstallCRDs      bool
	Version          string
	RecipeVersion    string
}

// GenerateBundleMetadata creates bundle metadata from config map.
func GenerateBundleMetadata(config map[string]string) *BundleMetadata {
	data := &BundleMetadata{
		Namespace:        common.GetConfigValue(config, "namespace", Name),
		HelmRepository:   common.GetConfigValue(config, "helm_repository", "https://charts.jetstack.io"),
		HelmChart:        "jetstack/cert-manager",
		HelmChartVersion: common.GetConfigValue(config, "helm_chart_version", "v1.19.1"),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(config),
		InstallCRDs:      true,
	}

	return data
}
