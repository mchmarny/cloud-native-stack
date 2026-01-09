package nvsentinel

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// HelmValues represents the data structure for NVSentinel Helm values.
type HelmValues struct {
	Timestamp         string
	Version           string
	RecipeVersion     string
	Namespace         string
	NVSentinelVersion common.ValueWithContext
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		Namespace:         common.GetConfigValue(config, "namespace", Name),
		Version:           common.GetBundlerVersion(config),
		RecipeVersion:     common.GetRecipeBundlerVersion(recipe.Metadata),
		NVSentinelVersion: common.ValueWithContext{Value: common.GetConfigValue(config, "nvsentinel_version", "v0.6.0")},
	}

	// Extract NVSentinel-specific settings from recipe measurements
	for _, m := range recipe.Measurements {
		switch m.Type {
		case measurement.TypeK8s:
			values.extractK8sSettings(m)
		case measurement.TypeGPU, measurement.TypeOS, measurement.TypeSystemD:
			// Not used by NVSentinel Helm values
		}
	}

	// Apply config overrides
	values.applyConfigOverrides(config)

	return values
}

// extractK8sSettings extracts Kubernetes-related settings from measurements.
func (v *HelmValues) extractK8sSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		subtypeContext := common.GetSubtypeContext(st.Context)

		// Extract version information from 'image' subtype
		if st.Name == "image" {
			if val, ok := st.Data["nvsentinel"]; ok {
				if s, ok := val.Any().(string); ok {
					v.NVSentinelVersion = common.ValueWithContext{
						Value:   s,
						Context: subtypeContext,
					}
				}
			}
		}
	}
}

// applyConfigOverrides applies configuration overrides.
func (v *HelmValues) applyConfigOverrides(config map[string]string) {
	if val := common.GetConfigValue(config, "nvsentinel_version", ""); val != "" {
		v.NVSentinelVersion = common.ValueWithContext{Value: val}
	}
}
