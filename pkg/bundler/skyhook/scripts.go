package skyhook

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// ScriptData represents the data structure for installation scripts.
type ScriptData struct {
	Timestamp        string
	Version          string
	RecipeVersion    string
	Namespace        string
	HelmChartRepo    common.ValueWithContext
	HelmChartName    common.ValueWithContext
	HelmReleaseName  common.ValueWithContext
	OperatorRegistry common.ValueWithContext
}

// GenerateScriptData generates script data from a recipe.
func GenerateScriptData(recipe *recipe.Recipe, config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(recipe.Metadata),
		Namespace:        common.GetConfigValue(config, "namespace", Name),
		HelmChartRepo:    common.ValueWithContext{Value: common.GetConfigValue(config, "helm_chart_repo", "https://nvidia.github.io/skyhook")},
		HelmChartName:    common.ValueWithContext{Value: common.GetConfigValue(config, "helm_chart_name", "skyhook")},
		HelmReleaseName:  common.ValueWithContext{Value: common.GetConfigValue(config, "helm_release_name", "skyhook")},
		OperatorRegistry: common.ValueWithContext{Value: common.GetConfigValue(config, "operator_registry", "nvcr.io/nvidia")},
	}

	return data
}
