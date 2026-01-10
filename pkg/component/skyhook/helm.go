package skyhook

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// HelmValues represents the data structure for Skyhook Helm values.
type HelmValues struct {
	Timestamp              string
	Version                string
	RecipeVersion          string
	Namespace              string
	OperatorRegistry       string
	KubeRbacProxyVersion   string
	SkyhookOperatorVersion string
	SkyhookAgentImage      string
	ManagerCPULimit        string
	ManagerMemoryLimit     string
	ManagerCPURequest      string
	ManagerMemoryRequest   string
	NodeSelector           string
	TolerationKey          string
	TolerationValue        string
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string, overrides map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:              time.Now().UTC().Format(time.RFC3339),
		Namespace:              common.GetConfigValue(config, "namespace", Name),
		Version:                common.GetBundlerVersion(config),
		RecipeVersion:          common.GetRecipeBundlerVersion(config),
		OperatorRegistry:       common.GetConfigValue(config, "operator_registry", "nvcr.io/nvidia"),
		KubeRbacProxyVersion:   common.GetConfigValue(config, "kube_rbac_proxy_version", "v0.15.0"),
		SkyhookOperatorVersion: common.GetConfigValue(config, "skyhook_operator_version", "v0.7.4"),
		SkyhookAgentImage:      common.GetConfigValue(config, "skyhook_agent_image", "nvcr.io/nvidia/skyhook-agent:latest"),
		ManagerCPULimit:        common.GetConfigValue(config, "manager_cpu_limit", "1000m"),
		ManagerMemoryLimit:     common.GetConfigValue(config, "manager_memory_limit", "4000Mi"),
		ManagerCPURequest:      common.GetConfigValue(config, "manager_cpu_request", "1000m"),
		ManagerMemoryRequest:   common.GetConfigValue(config, "manager_memory_request", "2000Mi"),
		NodeSelector:           common.GetConfigValue(config, "node_selector", "dedicated"),
		TolerationKey:          common.GetConfigValue(config, "toleration_key", "dedicated"),
		TolerationValue:        common.GetConfigValue(config, "toleration_value", "system-workload"),
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

	// Apply value overrides from --set flags
	values.applyValueOverrides(overrides)

	return values
}

// GenerateHelmValuesFromMap generates Helm values from config map (for RecipeResult inputs).
func GenerateHelmValuesFromMap(config map[string]string) *HelmValues {
	helmValues := &HelmValues{
		Timestamp:              time.Now().UTC().Format(time.RFC3339),
		Namespace:              common.GetConfigValue(config, "namespace", Name),
		Version:                common.GetBundlerVersion(config),
		RecipeVersion:          common.GetRecipeBundlerVersion(config),
		OperatorRegistry:       common.GetConfigValue(config, "operator_registry", "nvcr.io/nvidia"),
		KubeRbacProxyVersion:   common.GetConfigValue(config, "kube_rbac_proxy_version", "v0.15.0"),
		SkyhookOperatorVersion: common.GetConfigValue(config, "helm_chart_version", "v0.7.4"),
		SkyhookAgentImage:      common.GetConfigValue(config, "skyhook_agent_image", "nvcr.io/nvidia/skyhook-agent:latest"),
		ManagerCPULimit:        common.GetConfigValue(config, "manager_cpu_limit", "1000m"),
		ManagerMemoryLimit:     common.GetConfigValue(config, "manager_memory_limit", "4000Mi"),
		ManagerCPURequest:      common.GetConfigValue(config, "manager_cpu_request", "1000m"),
		ManagerMemoryRequest:   common.GetConfigValue(config, "manager_memory_request", "2000Mi"),
		NodeSelector:           common.GetConfigValue(config, "node_selector", "dedicated"),
		TolerationKey:          common.GetConfigValue(config, "toleration_key", "dedicated"),
		TolerationValue:        common.GetConfigValue(config, "toleration_value", "system-workload"),
	}

	return helmValues
}

// extractK8sSettings extracts Kubernetes-related settings from measurements.
func (v *HelmValues) extractK8sSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		// Extract version information from 'image' subtype
		if st.Name == "image" {
			if val, ok := st.Data["skyhook-operator"]; ok {
				if s, ok := val.Any().(string); ok {
					v.SkyhookOperatorVersion = s
				}
			}
			if val, ok := st.Data["skyhook-agent"]; ok {
				if s, ok := val.Any().(string); ok {
					v.SkyhookAgentImage = s
				}
			}
			if val, ok := st.Data["kube-rbac-proxy"]; ok {
				if s, ok := val.Any().(string); ok {
					v.KubeRbacProxyVersion = s
				}
			}
		}

		// Extract registry information
		if st.Name == "registry" {
			if val, ok := st.Data["uri"]; ok {
				if s, ok := val.Any().(string); ok {
					v.OperatorRegistry = s
				}
			}
		}

		// Extract configuration settings
		if st.Name == "config" {
			if val, ok := st.Data["manager-cpu-limit"]; ok {
				if s, ok := val.Any().(string); ok {
					v.ManagerCPULimit = s
				}
			}
			if val, ok := st.Data["manager-memory-limit"]; ok {
				if s, ok := val.Any().(string); ok {
					v.ManagerMemoryLimit = s
				}
			}
			if val, ok := st.Data["manager-cpu-request"]; ok {
				if s, ok := val.Any().(string); ok {
					v.ManagerCPURequest = s
				}
			}
			if val, ok := st.Data["manager-memory-request"]; ok {
				if s, ok := val.Any().(string); ok {
					v.ManagerMemoryRequest = s
				}
			}
		}
	}
}

// applyConfigOverrides applies configuration overrides.
func (v *HelmValues) applyConfigOverrides(config map[string]string) {
	if val := common.GetConfigValue(config, "skyhook_operator_version", ""); val != "" {
		v.SkyhookOperatorVersion = val
	}
	if val := common.GetConfigValue(config, "kube_rbac_proxy_version", ""); val != "" {
		v.KubeRbacProxyVersion = val
	}
	if val := common.GetConfigValue(config, "operator_registry", ""); val != "" {
		v.OperatorRegistry = val
	}
}

// applyValueOverrides applies value overrides from --set flags.
func (v *HelmValues) applyValueOverrides(overrides map[string]string) {
	if overrides == nil {
		return
	}

	for path, value := range overrides {
		switch path {
		case "operator.registry":
			v.OperatorRegistry = value
		case "operator.version":
			v.SkyhookOperatorVersion = value
		case "kubeRbacProxy.version":
			v.KubeRbacProxyVersion = value
		case "agent.image":
			v.SkyhookAgentImage = value
		case "manager.resources.cpu.limit":
			v.ManagerCPULimit = value
		case "manager.resources.memory.limit":
			v.ManagerMemoryLimit = value
		case "manager.resources.cpu.request":
			v.ManagerCPURequest = value
		case "manager.resources.memory.request":
			v.ManagerMemoryRequest = value
		case "nodeSelector":
			v.NodeSelector = value
		case "tolerations.key":
			v.TolerationKey = value
		case "tolerations.value":
			v.TolerationValue = value
		case "namespace":
			v.Namespace = value
		}
	}
}
