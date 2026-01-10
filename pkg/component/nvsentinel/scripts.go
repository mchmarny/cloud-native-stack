package nvsentinel

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// ScriptData represents the data structure for installation scripts.
type ScriptData struct {
	Timestamp         string
	Version           string
	RecipeVersion     string
	Namespace         string
	HelmChartRepo     string
	HelmReleaseName   string
	NVSentinelVersion string
}

// GenerateScriptData generates script data from a recipe.
func GenerateScriptData(recipe *recipe.Recipe, config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		Version:           common.GetBundlerVersion(config),
		RecipeVersion:     common.GetRecipeBundlerVersion(recipe.Metadata),
		Namespace:         common.GetConfigValue(config, "namespace", Name),
		HelmChartRepo:     common.GetConfigValue(config, "helm_chart_repo", "oci://ghcr.io/nvidia/nvsentinel"),
		HelmReleaseName:   common.GetConfigValue(config, "helm_release_name", "nvsentinel"),
		NVSentinelVersion: common.GetConfigValue(config, "nvsentinel_version", "v0.6.0"),
	}

	// Extract NVSentinel configuration from recipe measurements
	for _, m := range recipe.Measurements {
		if m.Type == measurement.TypeK8s {
			data.extractK8sSettings(m)
		}
	}

	return data
}

// GenerateScriptDataFromConfig creates script data from config map only (for RecipeResult inputs).
func GenerateScriptDataFromConfig(config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		Version:           common.GetBundlerVersion(config),
		RecipeVersion:     common.GetRecipeBundlerVersion(config),
		Namespace:         common.GetConfigValue(config, "namespace", Name),
		HelmChartRepo:     common.GetConfigValue(config, "helm_repository", "oci://ghcr.io/nvidia/nvsentinel"),
		HelmReleaseName:   common.GetConfigValue(config, "helm_release_name", "nvsentinel"),
		NVSentinelVersion: common.GetConfigValue(config, "helm_chart_version", "v0.6.0"),
	}

	return data
}

// extractK8sSettings extracts Kubernetes-related settings from measurements.
func (s *ScriptData) extractK8sSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		// Extract configuration from 'nvsentinel-config' subtype
		if st.Name == "nvsentinel-config" {
			if val, ok := st.Data["helm_chart_repo"]; ok {
				if repoStr, ok := val.Any().(string); ok {
					s.HelmChartRepo = repoStr
				}
			}
			if val, ok := st.Data["helm_release_name"]; ok {
				if releaseStr, ok := val.Any().(string); ok {
					s.HelmReleaseName = releaseStr
				}
			}
		}
	}
}
