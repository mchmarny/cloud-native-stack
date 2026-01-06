package skyhook

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// HelmValues represents the data structure for Skyhook Helm values.
type HelmValues struct {
	Timestamp              string
	Version                string
	RecipeVersion          string
	Namespace              string
	OperatorRegistry       common.ValueWithContext
	KubeRbacProxyVersion   common.ValueWithContext
	SkyhookOperatorVersion common.ValueWithContext
	SkyhookAgentImage      common.ValueWithContext
	ManagerCPULimit        common.ValueWithContext
	ManagerMemoryLimit     common.ValueWithContext
	ManagerCPURequest      common.ValueWithContext
	ManagerMemoryRequest   common.ValueWithContext
	NodeSelector           common.ValueWithContext
	TolerationKey          common.ValueWithContext
	TolerationValue        common.ValueWithContext
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:              time.Now().UTC().Format(time.RFC3339),
		Namespace:              common.GetConfigValue(config, "namespace", Name),
		Version:                common.GetBundlerVersion(config),
		RecipeVersion:          common.GetRecipeBundlerVersion(recipe.Metadata),
		OperatorRegistry:       common.ValueWithContext{Value: common.GetConfigValue(config, "operator_registry", "nvcr.io/nvidia")},
		KubeRbacProxyVersion:   common.ValueWithContext{Value: common.GetConfigValue(config, "kube_rbac_proxy_version", "v0.15.0")},
		SkyhookOperatorVersion: common.ValueWithContext{Value: common.GetConfigValue(config, "skyhook_operator_version", "v0.7.4")},
		SkyhookAgentImage:      common.ValueWithContext{Value: common.GetConfigValue(config, "skyhook_agent_image", "nvcr.io/nvidia/skyhook-agent:latest")},
		ManagerCPULimit:        common.ValueWithContext{Value: common.GetConfigValue(config, "manager_cpu_limit", "1000m")},
		ManagerMemoryLimit:     common.ValueWithContext{Value: common.GetConfigValue(config, "manager_memory_limit", "4000Mi")},
		ManagerCPURequest:      common.ValueWithContext{Value: common.GetConfigValue(config, "manager_cpu_request", "1000m")},
		ManagerMemoryRequest:   common.ValueWithContext{Value: common.GetConfigValue(config, "manager_memory_request", "2000Mi")},
		NodeSelector:           common.ValueWithContext{Value: common.GetConfigValue(config, "node_selector", "dedicated")},
		TolerationKey:          common.ValueWithContext{Value: common.GetConfigValue(config, "toleration_key", "dedicated")},
		TolerationValue:        common.ValueWithContext{Value: common.GetConfigValue(config, "toleration_value", "system-workload")},
	}

	// Extract Skyhook-specific settings from recipe measurements
	for _, m := range recipe.Measurements {
		switch m.Type {
		case measurement.TypeK8s:
			values.extractK8sSettings(m)
		case measurement.TypeGPU, measurement.TypeOS, measurement.TypeSystemD:
			// Not used by Skyhook Helm values
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
			if val, ok := st.Data["skyhook-operator"]; ok {
				if s, ok := val.Any().(string); ok {
					v.SkyhookOperatorVersion = common.ValueWithContext{
						Value:   s,
						Context: subtypeContext,
					}
				}
			}
			if val, ok := st.Data["skyhook-agent"]; ok {
				if s, ok := val.Any().(string); ok {
					v.SkyhookAgentImage = common.ValueWithContext{
						Value:   s,
						Context: subtypeContext,
					}
				}
			}
			if val, ok := st.Data["kube-rbac-proxy"]; ok {
				if s, ok := val.Any().(string); ok {
					v.KubeRbacProxyVersion = common.ValueWithContext{
						Value:   s,
						Context: subtypeContext,
					}
				}
			}
		}

		// Extract registry information
		if st.Name == "registry" {
			if val, ok := st.Data["uri"]; ok {
				if s, ok := val.Any().(string); ok {
					v.OperatorRegistry = common.ValueWithContext{
						Value:   s,
						Context: subtypeContext,
					}
				}
			}
		}

		// Extract configuration settings
		if st.Name == "config" {
			if val, ok := st.Data["manager-cpu-limit"]; ok {
				if s, ok := val.Any().(string); ok {
					v.ManagerCPULimit = common.ValueWithContext{
						Value:   s,
						Context: subtypeContext,
					}
				}
			}
			if val, ok := st.Data["manager-memory-limit"]; ok {
				if s, ok := val.Any().(string); ok {
					v.ManagerMemoryLimit = common.ValueWithContext{
						Value:   s,
						Context: subtypeContext,
					}
				}
			}
			if val, ok := st.Data["manager-cpu-request"]; ok {
				if s, ok := val.Any().(string); ok {
					v.ManagerCPURequest = common.ValueWithContext{
						Value:   s,
						Context: subtypeContext,
					}
				}
			}
			if val, ok := st.Data["manager-memory-request"]; ok {
				if s, ok := val.Any().(string); ok {
					v.ManagerMemoryRequest = common.ValueWithContext{
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
	if val := common.GetConfigValue(config, "skyhook_operator_version", ""); val != "" {
		v.SkyhookOperatorVersion = common.ValueWithContext{Value: val}
	}
	if val := common.GetConfigValue(config, "kube_rbac_proxy_version", ""); val != "" {
		v.KubeRbacProxyVersion = common.ValueWithContext{Value: val}
	}
	if val := common.GetConfigValue(config, "operator_registry", ""); val != "" {
		v.OperatorRegistry = common.ValueWithContext{Value: val}
	}
}
