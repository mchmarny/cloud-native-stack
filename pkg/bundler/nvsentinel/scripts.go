package nvsentinel

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
	HelmReleaseName  common.ValueWithContext
	NVSentinelVersion common.ValueWithContext
}

// GenerateScriptData generates script data from a recipe.
func GenerateScriptData(recipe *recipe.Recipe, config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(recipe.Metadata),
		Namespace:        common.GetConfigValue(config, "namespace", Name),
		HelmChartRepo:    common.ValueWithContext{Value: common.GetConfigValue(config, "helm_chart_repo", "oci://ghcr.io/nvidia/nvsentinel")},
		HelmReleaseName:  common.ValueWithContext{Value: common.GetConfigValue(config, "helm_release_name", "nvsentinel")},
		NVSentinelVersion: common.ValueWithContext{Value: common.GetConfigValue(config, "nvsentinel_version", "v0.6.0")},
	}

	return data
}
