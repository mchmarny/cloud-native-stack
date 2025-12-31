package networkoperator

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// ScriptData represents data for generating installation scripts.
type ScriptData struct {
	Timestamp        string
	Namespace        string
	HelmRepository   string
	HelmChart        string
	HelmChartVersion string
	K8sVersion       string
	EnableRDMA       bool
	EnableSRIOV      bool
	Request          *recipe.Query
	Version          string
	RecipeVersion    string
}

// GenerateScriptData creates script data from a recipe and config.
func GenerateScriptData(recipe *recipe.Recipe, config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Namespace:      common.GetConfigValue(config, "namespace", "nvidia-network-operator"),
		HelmRepository: common.GetConfigValue(config, "helm_repository", "https://helm.ngc.nvidia.com/nvidia"),
		HelmChart:      "nvidia/network-operator",
		Request:        recipe.Request,
		Version:        common.GetBundlerVersion(config),
		RecipeVersion:  common.GetRecipeBundlerVersion(recipe.Metadata),
	}

	// Extract chart version from config or recipe
	if val, ok := config["helm_chart_version"]; ok && val != "" {
		data.HelmChartVersion = val
	}

	// Extract feature flags
	if val, ok := config["enable_rdma"]; ok {
		data.EnableRDMA = val == "true"
	}
	if val, ok := config["enable_sriov"]; ok {
		data.EnableSRIOV = val == "true"
	}

	// Extract Kubernetes version from request
	if recipe.Request != nil && recipe.Request.K8s != nil {
		data.K8sVersion = recipe.Request.K8s.String()
	}

	return data
}

// ToMap converts ScriptData to a map for template rendering.
