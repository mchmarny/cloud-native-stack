package gpuoperator

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
	GPUType          string
	Request          *recipe.Query
	Version          string
	RecipeVersion    string
}

// GenerateScriptData creates script data from a recipe and config.
func GenerateScriptData(recipe *recipe.Recipe, config map[string]string) *ScriptData {
	data := &ScriptData{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Namespace:      common.GetConfigValue(config, "namespace", "gpu-operator"),
		HelmRepository: common.GetConfigValue(config, "helm_repository", "https://helm.ngc.nvidia.com/nvidia"),
		HelmChart:      "nvidia/gpu-operator",
		Request:        recipe.Request,
		Version:        common.GetBundlerVersion(config),
		RecipeVersion:  common.GetRecipeBundlerVersion(recipe.Metadata),
	}

	// Extract chart version from config or recipe
	if val, ok := config["helm_chart_version"]; ok && val != "" {
		data.HelmChartVersion = val
	}

	// Extract GPU type from request
	if recipe.Request != nil {
		data.GPUType = recipe.Request.GPU.String()
		if recipe.Request.K8s != nil {
			data.K8sVersion = recipe.Request.K8s.String()
		}
	}

	return data
}

// ToMap converts ScriptData to a map for template rendering.
func (s *ScriptData) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"Timestamp":        s.Timestamp,
		"Namespace":        s.Namespace,
		"HelmRepository":   s.HelmRepository,
		"HelmChart":        s.HelmChart,
		"HelmChartVersion": s.HelmChartVersion,
		"K8sVersion":       s.K8sVersion,
		"GPUType":          s.GPUType,
		"Version":          s.Version,
		"RecipeVersion":    s.RecipeVersion,
	}

	if s.Request != nil {
		m["Request"] = map[string]interface{}{
			"Os":         s.Request.Os.String(),
			"OsVersion":  s.Request.OsVersionString(),
			"Kernel":     s.Request.K8sString(),
			"Kubernetes": s.Request.K8sString(),
			"Service":    s.Request.Service.String(),
			"GPU":        s.Request.GPU.String(),
			"Intent":     s.Request.Intent.String(),
		}
	}

	return m
}
