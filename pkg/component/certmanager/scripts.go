package certmanager

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// ScriptData represents data for generating installation scripts and documentation.
type ScriptData struct {
	Timestamp        string
	Namespace        string
	HelmRepository   string
	HelmChart        string
	HelmChartVersion string
	InstallCRDs      bool
	Request          *recipe.RequestInfo
	Version          string
	RecipeVersion    string
}

// GenerateScriptData creates script data from a recipe and config.
func GenerateScriptData(recipe *recipe.Recipe, config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Namespace:      common.GetConfigValue(config, "namespace", Name),
		HelmRepository: common.GetConfigValue(config, "helm_repository", "https://charts.jetstack.io"),
		HelmChart:      "jetstack/cert-manager",
		Request:        recipe.Request,
		Version:        common.GetBundlerVersion(config),
		RecipeVersion:  common.GetRecipeBundlerVersion(recipe.Metadata),
		InstallCRDs:    true,
	}

	// Extract chart version from config or recipe
	if val, ok := config["helm_chart_version"]; ok && val != "" {
		data.HelmChartVersion = val
	} else if val, ok := config["cert_manager_version"]; ok && val != "" {
		data.HelmChartVersion = val
	} else {
		// Default to v1.19.1 if not specified
		data.HelmChartVersion = "v1.19.1"
	}

	// Extract CRD installation flag
	if val, ok := config["install_crds"]; ok {
		data.InstallCRDs = val == "true" || val == "1"
	}

	return data
}

// GenerateScriptDataFromConfig creates script data from config map only (for RecipeResult inputs).
func GenerateScriptDataFromConfig(config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
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
