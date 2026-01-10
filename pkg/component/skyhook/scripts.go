package skyhook

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// ScriptData represents the data structure for installation scripts.
type ScriptData struct {
	Timestamp        string
	Version          string
	RecipeVersion    string
	Namespace        string
	HelmChartRepo    string
	HelmChartName    string
	HelmReleaseName  string
	OperatorRegistry string
}

// GenerateScriptData generates script data from a recipe.
func GenerateScriptData(recipe *recipe.Recipe, config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(recipe.Metadata),
		Namespace:        common.GetConfigValue(config, "namespace", Name),
		HelmChartRepo:    common.GetConfigValue(config, "helm_chart_repo", "https://nvidia.github.io/skyhook"),
		HelmChartName:    common.GetConfigValue(config, "helm_chart_name", "skyhook"),
		HelmReleaseName:  common.GetConfigValue(config, "helm_release_name", "skyhook"),
		OperatorRegistry: common.GetConfigValue(config, "operator_registry", "nvcr.io/nvidia"),
	}

	return data
}

// GenerateScriptDataFromConfig creates script data from config map only (for RecipeResult inputs).
func GenerateScriptDataFromConfig(config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
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
