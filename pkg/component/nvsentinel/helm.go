package nvsentinel

import (
	"fmt"
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// HelmValues represents the data structure for NVSentinel Helm values.
type HelmValues struct {
	Timestamp         string
	Version           string
	RecipeVersion     string
	Namespace         string
	NVSentinelVersion string
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string, overrides map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		Namespace:         common.GetConfigValue(config, "namespace", Name),
		Version:           common.GetBundlerVersion(config),
		RecipeVersion:     common.GetRecipeBundlerVersion(config),
		NVSentinelVersion: common.GetConfigValue(config, "nvsentinel_version", "v0.6.0"),
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

	// Apply value overrides from --set flags
	values.applyValueOverrides(overrides)

	return values
}

// GenerateHelmValuesFromMap generates Helm values from config map (for RecipeResult inputs).
func GenerateHelmValuesFromMap(config map[string]string) *HelmValues {
	helmValues := &HelmValues{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		Namespace:         common.GetConfigValue(config, "namespace", Name),
		Version:           common.GetBundlerVersion(config),
		RecipeVersion:     common.GetRecipeBundlerVersion(config),
		NVSentinelVersion: common.GetConfigValue(config, "helm_chart_version", "v0.6.0"),
	}

	return helmValues
}

// extractK8sSettings extracts Kubernetes-related settings from measurements.
func (v *HelmValues) extractK8sSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		// Extract version information from 'image' subtype
		if st.Name == "image" {
			if val, ok := st.Data["nvsentinel"]; ok {
				if s, ok := val.Any().(string); ok {
					v.NVSentinelVersion = s
				}
			}
		}
	}
}

// applyConfigOverrides applies configuration overrides.
func (v *HelmValues) applyConfigOverrides(config map[string]string) {
	if val := common.GetConfigValue(config, "nvsentinel_version", ""); val != "" {
		v.NVSentinelVersion = val
	}
}

// applyValueOverrides applies value overrides from --set flags.
// Uses reflection-based approach to dynamically set any field using dot notation.
func (v *HelmValues) applyValueOverrides(overrides map[string]string) {
	if len(overrides) == 0 {
		return
	}

	// Use reflection-based override utility for dynamic field setting
	if err := common.ApplyValueOverrides(v, overrides); err != nil {
		// Log error but continue - some overrides may have succeeded
		fmt.Printf("Warning: failed to apply some value overrides: %v\n", err)
	}
}
